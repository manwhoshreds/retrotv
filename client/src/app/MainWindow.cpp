#include <QApplication>
#include <QByteArray>
#include <QDateTime>
#include <QEvent>
#include <QEventLoop>
#include <QFutureWatcher>
#include <QFrame>
#include <QHBoxLayout>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonValue>
#include <QKeyEvent>
#include <QLabel>
#include <QMoveEvent>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QNetworkRequest>
#include <QPainter>
#include <QPixmap>
#include <QPushButton>
#include <QDebug>
#include <QResizeEvent>
#include <QStackedLayout>
#include <QString>
#include <QStatusBar>
#include <QThread>
#include <QTimer>
#include <QtConcurrent>
#include <QUrl>
#include <QVBoxLayout>

#include "MainWindow.h"
#include "app/ChannelTransitionOverlay.h"
#include "app/ClientSettings.h"
#include "app/GuideOverlay.h"
#include "app/InfoOverlay.h"
#include "app/SettingsPage.h"
#include "player/PlayerWidget.h"
#include "network/ServerDiscovery.h"
#include "xmltv/XMLTVParser.h"

namespace {

// Buttons sent by TV-style remotes. On eglfs, Qt has no key for KEY_SELECT
// (the remote's OK button), so it is matched by scancode: evdev code 353
// plus the usual offset of 8.
constexpr quint32 kRemoteSelectScanCode = 353 + 8;

bool isRemoteSelectKey(const QKeyEvent *event)
{
    return event->key() == Qt::Key_Select
        || (event->key() == 0 && event->nativeScanCode() == kRemoteSelectScanCode);
}

struct ChannelLoadResult
{
    bool connected = false;
    std::vector<Channel> channels;
};

struct StartupLoadResult
{
    ChannelLoadResult channelResult;
    XMLTVGuide guide;
    QString backendURL;
    QString discoveredURL;
    QList<DiscoveredServer> discoveredServers;
};

QString publicURL(const QString &baseURL, const QString &path)
{
    if (path.isEmpty() || path.startsWith(QStringLiteral("http://")) || path.startsWith(QStringLiteral("https://"))) {
        return path;
    }
    if (path.startsWith(QLatin1Char('/'))) {
        return baseURL + path;
    }
    return path;
}

std::optional<std::string> fetchText(const QUrl &url, int timeoutMS = 3000)
{
    QNetworkAccessManager manager;
    QNetworkRequest request(url);
    request.setHeader(QNetworkRequest::UserAgentHeader, QStringLiteral("retrotv-client/0.1"));
    request.setTransferTimeout(timeoutMS);

    QNetworkReply *reply = manager.get(request);
    QEventLoop loop;
    QObject::connect(reply, &QNetworkReply::finished, &loop, &QEventLoop::quit);
    loop.exec();

    if (reply->error() != QNetworkReply::NoError) {
        qWarning() << "RetroTV request failed" << url << reply->errorString();
        reply->deleteLater();
        return std::nullopt;
    }

    const int statusCode = reply->attribute(QNetworkRequest::HttpStatusCodeAttribute).toInt();
    if (statusCode >= 400) {
        const QByteArray body = reply->readAll();
        qWarning() << "RetroTV request returned HTTP" << statusCode << url << QString::fromUtf8(body).left(500);
        reply->deleteLater();
        return std::nullopt;
    }

    const QByteArray body = reply->readAll();
    reply->deleteLater();
    return std::string(body.constData(), static_cast<std::size_t>(body.size()));
}

QString formatProgramTimeRange(const QDateTime &start, const QDateTime &stop)
{
    if (!start.isValid() || !stop.isValid()) {
        return {};
    }
    return QStringLiteral("%1 - %2")
        .arg(start.toLocalTime().time().toString(QStringLiteral("h:mm AP")),
             stop.toLocalTime().time().toString(QStringLiteral("h:mm AP")));
}

QString titleWithYear(const QJsonObject &media)
{
    const QString title = media.value(QStringLiteral("title")).toString();
    const int year = media.value(QStringLiteral("year")).toInt();
    if (title.isEmpty() || year <= 0 || title.endsWith(QStringLiteral(")"))) {
        return title;
    }
    return QStringLiteral("%1 (%2)").arg(title).arg(year);
}

QString episodeCode(int seasonNumber, int episodeNumber)
{
    if (seasonNumber <= 0 || episodeNumber <= 0) {
        return {};
    }
    return QStringLiteral("S%1E%2")
        .arg(seasonNumber, 2, 10, QLatin1Char('0'))
        .arg(episodeNumber, 2, 10, QLatin1Char('0'));
}

QString mediaDisplayTitle(const QJsonObject &media)
{
    if (media.value(QStringLiteral("type")).toString() != QStringLiteral("tv")) {
        return titleWithYear(media);
    }

    const QString showTitle = media.value(QStringLiteral("show_title")).toString(
        media.value(QStringLiteral("title")).toString());
    const QString code = episodeCode(
        media.value(QStringLiteral("season_number")).toInt(),
        media.value(QStringLiteral("episode_number")).toInt());
    const QString episodeTitle = media.value(QStringLiteral("episode_title")).toString();

    QString title = showTitle;
    if (!code.isEmpty()) {
        title = title.isEmpty() ? code : QStringLiteral("%1 %2").arg(title, code);
    }
    if (!episodeTitle.isEmpty()) {
        title = title.isEmpty() ? episodeTitle : QStringLiteral("%1 - %2").arg(title, episodeTitle);
    }
    return title.isEmpty() ? media.value(QStringLiteral("title")).toString() : title;
}

ChannelLoadResult loadChannels(const QString &baseURL)
{
    if (auto body = fetchText(QUrl(baseURL + QStringLiteral("/api/channels?airing=true")))) {
        const QJsonDocument document = QJsonDocument::fromJson(QByteArray::fromStdString(*body));
        const QJsonArray channelValues = document.object().value(QStringLiteral("channels")).toArray();
        std::vector<Channel> channels;
        int channelNumber = 1;
        for (const QJsonValue &channelValue : channelValues) {
            const QJsonObject channelObject = channelValue.toObject();
            const int id = channelObject.value(QStringLiteral("id")).toInt();
            const QString name = channelObject.value(QStringLiteral("name")).toString();
            const QString logoPath = channelObject.value(QStringLiteral("logo_path")).toString();
            if (id <= 0 || name.isEmpty()) {
                continue;
            }
            channels.push_back({
                id,
                channelNumber,
                name.toStdString(),
                QStringLiteral("%1/channels/%2/m3u").arg(baseURL).arg(id).toStdString(),
                publicURL(baseURL, logoPath).toStdString(),
            });
            ++channelNumber;
        }
        return {true, channels};
    }

    return {};
}

std::optional<PlaybackTarget> playbackTargetFromResponse(const QUrl &tuneURL, const QByteArray &body)
{
    const QJsonDocument document = QJsonDocument::fromJson(body);
    const QJsonObject response = document.object();
    const QJsonObject program = response.value(QStringLiteral("program")).toObject();
    if (program.isEmpty()) {
        return std::nullopt;
    }
    const QString streamURL = program.value(QStringLiteral("play_url")).toString();
    if (streamURL.isEmpty()) {
        qWarning() << "Tune response did not include play_url" << tuneURL << QString::fromUtf8(body).left(500);
        return std::nullopt;
    }
    qInfo() << "RetroTV tune target" << tuneURL << "play_url" << streamURL << "seek" << program.value(QStringLiteral("seek_position")).toInt();
    const QDateTime start = QDateTime::fromString(program.value(QStringLiteral("start_time")).toString(), Qt::ISODate).toUTC();
    const QDateTime stop = QDateTime::fromString(program.value(QStringLiteral("end_time")).toString(), Qt::ISODate).toUTC();
    const QJsonObject nextProgram = response.value(QStringLiteral("next_program")).toObject();
    const QDateTime nextStart = QDateTime::fromString(nextProgram.value(QStringLiteral("start_time")).toString(), Qt::ISODate).toUTC();
    const QDateTime nextStop = QDateTime::fromString(nextProgram.value(QStringLiteral("end_time")).toString(), Qt::ISODate).toUTC();
	const QJsonArray genres = program.value(QStringLiteral("genres")).toArray();
	QStringList genreNames;
	for (const QJsonValue &genre : genres) {
		if (!genre.toString().isEmpty()) {
			genreNames.append(genre.toString());
		}
	}
	const QJsonValue closedCaptions = program.value(QStringLiteral("has_closed_captions"));
	if (closedCaptions.isUndefined()) {
		qWarning() << "Tune response is missing has_closed_captions; restart or update the RetroTV backend";
	}
	const QJsonValue isHD = program.value(QStringLiteral("is_hd"));
	if (isHD.isUndefined()) {
		qWarning() << "Tune response is missing is_hd; restart or update the RetroTV backend";
	}

    return PlaybackTarget{
        streamURL,
        program.value(QStringLiteral("seek_position")).toInt(),
        program.value(QStringLiteral("title")).toString(),
        program.value(QStringLiteral("description")).toString(),
        program.value(QStringLiteral("poster_url")).toString(),
        formatProgramTimeRange(start, stop),
        nextProgram.value(QStringLiteral("title")).toString(),
        formatProgramTimeRange(nextStart, nextStop),
		start,
		stop,
		program.value(QStringLiteral("year")).toInt(),
		program.value(QStringLiteral("rating")).toString(),
		genreNames.join(QStringLiteral(", ")),
		program.value(QStringLiteral("season_number")).toInt(),
		program.value(QStringLiteral("episode_number")).toInt(),
		closedCaptions.toBool(),
		isHD.toBool(),
    };
}

XMLTVGuide loadGuide(const QString &baseURL, const std::vector<Channel> &channels)
{
    if (auto body = fetchText(QUrl(baseURL + QStringLiteral("/guide.xml?hours=12")))) {
        return XMLTVParser::parseContent(*body);
    }

    XMLTVGuide guide;
    for (const Channel &channel : channels) {
        QString guideURL = QString::fromStdString(channel.mediaUrl);
        if (!guideURL.endsWith(QStringLiteral("/m3u"))) {
            continue;
        }
        guideURL.chop(4);
        guideURL.append(QStringLiteral("/xmltv?hours=12"));

        if (auto body = fetchText(QUrl(guideURL))) {
            XMLTVGuide channelGuide = XMLTVParser::parseContent(*body);
            guide.channels.insert(guide.channels.end(), channelGuide.channels.begin(), channelGuide.channels.end());
            guide.programmes.insert(guide.programmes.end(), channelGuide.programmes.begin(), channelGuide.programmes.end());
        }
    }
    return guide;
}

}

MainWindow::MainWindow(QWidget *parent)
    : QMainWindow(parent)
{
    setFocusPolicy(Qt::StrongFocus);
	m_tuneNetworkManager = new QNetworkAccessManager(this);
    m_channelNumberTimer = new QTimer(this);
    m_channelNumberTimer->setSingleShot(true);
    m_channelNumberTimer->setInterval(1200);
    connect(m_channelNumberTimer, &QTimer::timeout, this, &MainWindow::commitChannelNumber);
    // The guide only covers the next 12 hours and schedules change when they
    // are published in the web UI, so keep it fresh while the client runs.
    m_guideRefreshTimer = new QTimer(this);
    m_guideRefreshTimer->setInterval(5 * 60 * 1000);
    connect(m_guideRefreshTimer, &QTimer::timeout, this, &MainWindow::refreshGuide);

    m_rootWidget = new QWidget(this);
    auto *rootLayout = new QStackedLayout(m_rootWidget);
    rootLayout->setContentsMargins(0, 0, 0, 0);
    rootLayout->setStackingMode(QStackedLayout::StackAll);

    m_playerWidget = new PlayerWidget(m_rootWidget);
    m_playerWidget->setFocusPolicy(Qt::NoFocus);
    connect(m_playerWidget, &PlayerWidget::mediaStatus, this, [this](const QString &message) {
        statusBar()->showMessage(message);
    });
    connect(m_playerWidget, &PlayerWidget::mediaLoaded, this, [this](const QString &) {
        const int generation = m_tuneGeneration;
        QTimer::singleShot(150, this, [this, generation]() {
            if (generation != m_tuneGeneration) {
                return;
            }
            if (m_channelTransitionOverlay) {
                m_channelTransitionOverlay->clearBlackoutWhenReady();
            }
            setStartupState(StartupState::Playing);
        });
    });
    connect(m_playerWidget, &PlayerWidget::mediaFailed, this, [this](const QString &, const QString &message) {
        if (m_channelTransitionOverlay) {
            m_channelTransitionOverlay->clearBlackoutWhenReady();
        }
        if (m_channels.empty()) {
            setStartupState(StartupState::MediaUnavailable, QStringLiteral("Media unavailable"), message);
            return;
        }
        showMediaUnavailable(m_channels.at(m_currentChannelIndex), message);
    });
    connect(m_playerWidget, &PlayerWidget::mediaEnded, this, [this](const QString &) {
        if (m_channels.empty()) {
            return;
        }
        tuneChannel(m_currentChannelIndex, false);
    });
    rootLayout->addWidget(m_playerWidget);

    m_stateView = new QWidget(m_rootWidget);
    m_stateView->setObjectName(QStringLiteral("stateView"));
    m_stateStack = new QStackedLayout(m_stateView);
    m_stateStack->setContentsMargins(0, 0, 0, 0);

    m_genericStateView = new QWidget(m_stateView);
    auto *genericLayout = new QVBoxLayout(m_genericStateView);
    genericLayout->setContentsMargins(48, 48, 48, 48);
    m_stateLabel = new QLabel(m_genericStateView);
    m_stateLabel->setAlignment(Qt::AlignCenter);
    m_stateLabel->setWordWrap(true);
    m_stateLabel->setTextFormat(Qt::RichText);
    genericLayout->addWidget(m_stateLabel);
    m_stateStack->addWidget(m_genericStateView);

    m_offlineView = new QWidget(m_stateView);
    m_offlineView->setObjectName(QStringLiteral("offlineView"));
    auto *offlineLayout = new QVBoxLayout(m_offlineView);
    offlineLayout->setContentsMargins(56, 50, 56, 24);
    offlineLayout->setSpacing(0);

    auto *headerLayout = new QHBoxLayout();
    headerLayout->setContentsMargins(0, 0, 0, 0);
    auto *brand = new QLabel(QStringLiteral(
        "<span style=\"color:#ff453a;\">━━</span>"
        "<span style=\"color:#ffd60a;\">━━</span>"
        "<span style=\"color:#30d158;\">━━</span>"
        "<span style=\"color:#0a84ff;\">━━</span>"
        "<span style=\"color:#f8fafc;\"> RETROTV</span>"),
        m_offlineView);
    brand->setObjectName(QStringLiteral("offlineBrand"));
    brand->setTextFormat(Qt::RichText);
    headerLayout->addWidget(brand);
    headerLayout->addStretch(1);
    auto *offlineStatus = new QLabel(QStringLiteral("● OFFLINE"), m_offlineView);
    offlineStatus->setObjectName(QStringLiteral("offlineStatus"));
    headerLayout->addWidget(offlineStatus);
    offlineLayout->addLayout(headerLayout);

    offlineLayout->addStretch(1);

    auto *icon = new QLabel(QStringLiteral("<div align=\"center\"><span style=\"color:#657386; font-size:58px;\">▔</span><br/><span style=\"color:#4b5a6f; font-size:78px;\">▣</span><br/><span style=\"color:#ff453a; font-size:42px; font-weight:900;\">!</span></div>"), m_offlineView);
    icon->setObjectName(QStringLiteral("offlineIcon"));
    icon->setTextFormat(Qt::RichText);
    icon->setAlignment(Qt::AlignCenter);
    offlineLayout->addWidget(icon);

    auto *offlineTitle = new QLabel(QStringLiteral("CAN'T CONNECT TO RETROTV SERVER"), m_offlineView);
    offlineTitle->setObjectName(QStringLiteral("offlineTitle"));
    offlineTitle->setAlignment(Qt::AlignCenter);
    offlineLayout->addWidget(offlineTitle);

    auto *offlineSubtitle = new QLabel(QStringLiteral("It looks like the RetroTV backend is not running\nor is unreachable."), m_offlineView);
    offlineSubtitle->setObjectName(QStringLiteral("offlineSubtitle"));
    offlineSubtitle->setAlignment(Qt::AlignCenter);
    offlineLayout->addWidget(offlineSubtitle);

    offlineLayout->addSpacing(50);

    auto *troubleHeader = new QHBoxLayout();
    auto *leftLine = new QFrame(m_offlineView);
    leftLine->setFrameShape(QFrame::HLine);
    leftLine->setObjectName(QStringLiteral("offlineRule"));
    auto *rightLine = new QFrame(m_offlineView);
    rightLine->setFrameShape(QFrame::HLine);
    rightLine->setObjectName(QStringLiteral("offlineRule"));
    auto *troubleTitle = new QLabel(QStringLiteral("TROUBLESHOOTING"), m_offlineView);
    troubleTitle->setObjectName(QStringLiteral("troubleTitle"));
    troubleTitle->setAlignment(Qt::AlignCenter);
    troubleHeader->addWidget(leftLine, 1);
    troubleHeader->addSpacing(26);
    troubleHeader->addWidget(troubleTitle);
    troubleHeader->addSpacing(26);
    troubleHeader->addWidget(rightLine, 1);
    offlineLayout->addLayout(troubleHeader);

    auto *troubleList = new QWidget(m_offlineView);
    auto *troubleLayout = new QVBoxLayout(troubleList);
    troubleLayout->setContentsMargins(0, 26, 0, 0);
    troubleLayout->setSpacing(20);
    const QList<QPair<QString, QString>> troubleItems = {
        {QStringLiteral("▰"), QStringLiteral("Make sure the RetroTV server is running.")},
        {QStringLiteral("<>"), QStringLiteral("Check your connection and try again.")},
        {QStringLiteral(">_"), QStringLiteral("Verify the server URL in your configuration.")},
    };
    for (const auto &item : troubleItems) {
        auto *row = new QHBoxLayout();
        row->setContentsMargins(0, 0, 0, 0);
        row->addStretch(1);
        auto *symbol = new QLabel(item.first, troubleList);
        symbol->setObjectName(QStringLiteral("troubleSymbol"));
        symbol->setAlignment(Qt::AlignCenter);
        symbol->setFixedWidth(58);
        auto *text = new QLabel(item.second, troubleList);
        text->setObjectName(QStringLiteral("troubleText"));
        row->addWidget(symbol);
        row->addSpacing(20);
        row->addWidget(text);
        row->addStretch(1);
        troubleLayout->addLayout(row);
    }
    offlineLayout->addWidget(troubleList);

    auto *buttonRow = new QWidget(m_stateView);
    auto *buttonLayout = new QHBoxLayout(buttonRow);
    buttonLayout->setContentsMargins(0, 30, 0, 0);
    buttonLayout->setSpacing(28);
    buttonLayout->addStretch(1);

    m_retryButton = new QPushButton(QStringLiteral("RETRY"), buttonRow);
    m_retryButton->setCursor(Qt::PointingHandCursor);
    connect(m_retryButton, &QPushButton::clicked, this, &MainWindow::startClient);
    buttonLayout->addWidget(m_retryButton);

    m_settingsButton = new QPushButton(QStringLiteral("CLIENT SETTINGS"), buttonRow);
    m_settingsButton->setObjectName(QStringLiteral("serverInfoButton"));
    m_settingsButton->setCursor(Qt::PointingHandCursor);
    connect(m_settingsButton, &QPushButton::clicked, this, &MainWindow::showSettings);
    buttonLayout->addWidget(m_settingsButton);
    buttonLayout->addStretch(1);
    offlineLayout->addWidget(buttonRow);

    offlineLayout->addStretch(1);

    auto *footer = new QFrame(m_offlineView);
    footer->setObjectName(QStringLiteral("offlineFooter"));
    auto *footerLayout = new QHBoxLayout(footer);
    footerLayout->setContentsMargins(20, 0, 20, 0);
    auto *footerText = new QLabel(QStringLiteral("ⓘ   Unable to reach RetroTV server at the configured address."), footer);
    footerText->setObjectName(QStringLiteral("footerText"));
    footerLayout->addWidget(footerText);
    footerLayout->addStretch(1);
    auto *version = new QLabel(QStringLiteral("v0.1.0"), footer);
    version->setObjectName(QStringLiteral("footerText"));
    footerLayout->addWidget(version);
    offlineLayout->addWidget(footer);
    m_stateStack->addWidget(m_offlineView);

    m_stateView->setStyleSheet(QStringLiteral(R"(
        QWidget#stateView {
            background: #040b12;
            color: #f8fafc;
        }
        QWidget#offlineView {
            background: #040b12;
        }
        QLabel {
            color: #f8fafc;
        }
        QLabel#offlineBrand {
            font-size: 42px;
            font-weight: 900;
            letter-spacing: 4px;
        }
        QLabel#offlineStatus {
            color: #ff453a;
            font-size: 24px;
            letter-spacing: 1px;
        }
        QLabel#offlineTitle {
            font-size: 42px;
            font-weight: 900;
            letter-spacing: 1px;
            margin-top: 28px;
        }
        QLabel#offlineSubtitle {
            color: #aebacc;
            font-size: 24px;
            line-height: 130%;
            margin-top: 22px;
        }
        QFrame#offlineRule {
            color: #26384d;
            background: #26384d;
            max-height: 1px;
        }
        QLabel#troubleTitle {
            color: #78869a;
            font-size: 22px;
            font-weight: 900;
        }
        QLabel#troubleSymbol {
            color: #aebacc;
            font-size: 22px;
            font-weight: 900;
        }
        QLabel#troubleText {
            color: #aebacc;
            font-size: 24px;
        }
        QPushButton {
            min-width: 260px;
            min-height: 56px;
            border-radius: 6px;
            border: 2px solid #2a3c55;
            background: transparent;
            color: #aebacc;
            font-size: 22px;
            font-weight: 800;
            letter-spacing: 1px;
        }
        QPushButton:hover {
            border-color: #4d6380;
            color: #f8fafc;
        }
        QPushButton#serverInfoButton {
            border-color: #1677ff;
            color: #4d96ff;
        }
        QFrame#offlineFooter {
            min-height: 74px;
            border: 1px solid #172539;
            border-radius: 8px;
            background: transparent;
        }
        QLabel#footerText {
            color: #aebacc;
            font-size: 19px;
        }
    )"));
    rootLayout->addWidget(m_stateView);

    m_infoOverlay = new InfoOverlay(m_rootWidget);
    connect(m_infoOverlay, &InfoOverlay::dismissRequested, this, &MainWindow::hideOverlays);
    m_infoOverlay->hide();

    m_guideOverlay = new GuideOverlay(m_rootWidget);
    m_guideOverlay->setTextSize(ClientSettings::guideTextSize(GuideOverlay::DefaultTextSize));
    connect(m_guideOverlay, &GuideOverlay::dismissRequested, this, &MainWindow::hideOverlays);
    connect(m_guideOverlay, &GuideOverlay::channelSelected, this, [this](int channelId) {
        tuneChannelId(channelId);
        hideOverlays();
    });
    m_guideOverlay->hide();
    qApp->installEventFilter(this);

    m_settingsPage = new SettingsPage(m_rootWidget);
    m_settingsPage->hide();
    connect(m_settingsPage, &SettingsPage::dismissRequested, this, &MainWindow::hideSettings);
    connect(m_settingsPage, &SettingsPage::guideTextSizeChanged, this, [this](int textSize) {
        m_guideOverlay->setTextSize(textSize);
        ClientSettings::setGuideTextSize(m_guideOverlay->textSize());
    });
    connect(m_settingsPage, &SettingsPage::serverSelected, this, [this](const QString &url) {
        m_requestedServerURL = url;
        hideSettings();
        startClient();
    });
    connect(m_settingsPage, &SettingsPage::saveRequested, this, [this](const ClientSettings &settings) {
        settings.save();
        hideSettings();
        m_channels.clear();
        m_currentChannelIndex = 0;
        ++m_tuneGeneration;
        if (m_tuneReply) {
            m_tuneReply->abort();
            m_tuneReply->deleteLater();
            m_tuneReply = nullptr;
        }
        startClient();
    });
    rootLayout->addWidget(m_settingsPage);

    m_channelTransitionOverlay = new ChannelTransitionOverlay(m_rootWidget);
    m_channelTransitionOverlay->hide();

    m_channelBugLabel = new QLabel(m_rootWidget);
    m_channelBugLabel->setObjectName(QStringLiteral("channelBug"));
    m_channelBugLabel->setAttribute(Qt::WA_TransparentForMouseEvents);
    m_channelBugLabel->setAutoFillBackground(false);
    m_channelBugLabel->setAlignment(Qt::AlignCenter);
    m_channelBugLabel->setScaledContents(false);
    m_channelBugLabel->setStyleSheet(QStringLiteral("QLabel#channelBug { background: transparent; border: none; }"));
    m_channelBugLabel->hide();

    setCentralWidget(m_rootWidget);
    updateOverlayGeometry();
    startClient();
    if (qApp->arguments().contains(QStringLiteral("--settings"))) {
        QTimer::singleShot(0, this, &MainWindow::showSettings);
    }
    if (qApp->arguments().contains(QStringLiteral("--settings-screenshot"))) {
        QTimer::singleShot(800, this, [this]() {
            grab().save(QStringLiteral("/tmp/retrotv-settings.png"));
            qApp->quit();
        });
    }
}

void MainWindow::showSettings()
{
    hideOverlays();
    m_settingsPage->loadSettings(ClientSettings::load(), m_discoveredServers, m_backendURL, m_backendConnected);
    m_settingsPage->setGuideTextSize(m_guideOverlay->textSize());
    m_settingsPage->show();
    m_settingsPage->raise();
}

void MainWindow::hideSettings()
{
    m_settingsPage->hide();
    setFocus(Qt::OtherFocusReason);
}

void MainWindow::startClient()
{
    const QString requestedServerURL = m_requestedServerURL;
    m_requestedServerURL.clear();
    m_backendConnected = false;
    setStartupState(
        StartupState::Starting,
        QStringLiteral("Starting RetroTV"),
        QStringLiteral("Looking for a RetroTV server on your network..."));

    auto *watcher = new QFutureWatcher<StartupLoadResult>(this);
    connect(watcher, &QFutureWatcher<StartupLoadResult>::finished, this, [this, watcher]() {
        const StartupLoadResult startup = watcher->result();
        watcher->deleteLater();
        m_backendURL = startup.backendURL;
        m_discoveredBackendURL = startup.discoveredURL;
        if (!startup.discoveredServers.isEmpty()) m_discoveredServers = startup.discoveredServers;
        m_backendConnected = startup.channelResult.connected;
        if (!startup.channelResult.connected) {
            setStartupState(StartupState::Offline, QStringLiteral("RetroTV is offline"),
                QStringLiteral("Discovery failed and the manually configured server at %1 could not be reached.").arg(m_backendURL));
            statusBar()->showMessage(QStringLiteral("Backend offline: %1").arg(m_backendURL));
            return;
        }
        m_channels = startup.channelResult.channels;
        if (m_channels.empty()) {
            setStartupState(StartupState::NoChannels, QStringLiteral("No channels configured"),
                QStringLiteral("Create channels in the RetroTV backend, then restart or retune the client."));
            statusBar()->showMessage(QStringLiteral("Backend connected, but no channels were returned."));
            return;
        }
        setStartupState(StartupState::LoadingGuide, QStringLiteral("Loading guide"),
            QStringLiteral("Preparing %1 channels.").arg(static_cast<int>(m_channels.size())));
        m_guideOverlay->setCurrentChannelId(m_channels.front().id);
        m_guideOverlay->setGuide(startup.guide);
        m_guideRefreshTimer->start();
        tuneChannel(0);
    });
    watcher->setFuture(QtConcurrent::run([requestedServerURL]() {
        StartupLoadResult startup;
        const QString environmentURL = QString::fromUtf8(qgetenv("RETROTV_BACKEND_URL")).trimmed();
        const QString manualURL = ClientSettings::load().backendURL;
        if (!requestedServerURL.isEmpty()) {
            startup.backendURL = requestedServerURL;
        } else {
            startup.discoveredServers = environmentURL.isEmpty() ? discoverRetroTVServers() : QList<DiscoveredServer>();
            startup.discoveredURL = startup.discoveredServers.isEmpty() ? QString() : startup.discoveredServers.front().url;
            startup.backendURL = environmentURL.isEmpty() ? startup.discoveredURL : environmentURL;
        }
        while (startup.backendURL.endsWith('/')) startup.backendURL.chop(1);
        if (!startup.backendURL.isEmpty()) {
            startup.channelResult = loadChannels(startup.backendURL);
        }
        const bool automatic = requestedServerURL.isEmpty() && environmentURL.isEmpty();
        const QString lastWorkingURL = ClientSettings::lastWorkingServerURL();
        if (!startup.channelResult.connected && automatic
            && !lastWorkingURL.isEmpty() && startup.backendURL != lastWorkingURL) {
            qWarning() << "RetroTV discovery found no reachable server; trying last working server" << lastWorkingURL;
            startup.backendURL = lastWorkingURL;
            startup.channelResult = loadChannels(startup.backendURL);
        }
        if (!startup.channelResult.connected && automatic && startup.backendURL != manualURL) {
            startup.backendURL = manualURL;
            while (startup.backendURL.endsWith('/')) startup.backendURL.chop(1);
            startup.channelResult = loadChannels(startup.backendURL);
        }
        if (startup.channelResult.connected) {
            ClientSettings::rememberWorkingServer(startup.backendURL);
        }
        if (startup.channelResult.connected && !startup.channelResult.channels.empty()) {
            startup.guide = loadGuide(startup.backendURL, startup.channelResult.channels);
        }
        return startup;
    }));
}

bool MainWindow::eventFilter(QObject *watched, QEvent *event)
{
    Q_UNUSED(watched);
    if (event->type() != QEvent::KeyPress) {
        return false;
    }

    auto *keyEvent = static_cast<QKeyEvent *>(event);
    if (m_settingsPage && m_settingsPage->isVisible()) {
        // Settings owns the remote while it is open: no channel changes or
        // number entry behind it, and digits still type into text fields.
        if (keyEvent->key() == Qt::Key_Escape || handleRemoteKey(keyEvent)) {
            if (keyEvent->key() == Qt::Key_Escape) {
                hideSettings();
            }
            keyEvent->accept();
            return true;
        }
        const bool select = isRemoteSelectKey(keyEvent)
            || keyEvent->key() == Qt::Key_Enter || keyEvent->key() == Qt::Key_Return;
        if (m_settingsPage->handleNavigationKey(keyEvent, select)) {
            keyEvent->accept();
            return true;
        }
        return false;
    }
    if (handleChannelNumberKey(keyEvent)) {
        keyEvent->accept();
        return true;
    }
    if (handleRemoteKey(keyEvent)) {
        keyEvent->accept();
        return true;
    }
    if (!m_guideOverlay || !m_guideOverlay->isVisible()) {
        return false;
    }

    switch (keyEvent->key()) {
    case Qt::Key_Escape:
        hideOverlays();
        keyEvent->accept();
        return true;
    case Qt::Key_Enter:
    case Qt::Key_Return:
        tuneChannelId(m_guideOverlay->selectedChannelId());
        hideOverlays();
        keyEvent->accept();
        return true;
    case Qt::Key_Up:
        m_guideOverlay->scrollChannels(-1);
        keyEvent->accept();
        return true;
    case Qt::Key_Down:
        m_guideOverlay->scrollChannels(1);
        keyEvent->accept();
        return true;
    case Qt::Key_Right:
        m_guideOverlay->scrollTimeline(1);
        keyEvent->accept();
        return true;
    case Qt::Key_Left:
        m_guideOverlay->scrollTimeline(-1);
        keyEvent->accept();
        return true;
    default:
        return false;
    }
}

bool MainWindow::handleRemoteKey(QKeyEvent *event)
{
    // Volume works everywhere, including over the guide and settings.
    switch (event->key()) {
    case Qt::Key_VolumeUp:
        m_playerWidget->adjustVolume(5);
        return true;
    case Qt::Key_VolumeDown:
        m_playerWidget->adjustVolume(-5);
        return true;
    case Qt::Key_VolumeMute:
        m_playerWidget->toggleMute();
        return true;
    default:
        break;
    }

    const bool isSettingsKey = event->key() == Qt::Key_Menu || event->key() == Qt::Key_Settings;
    if (m_settingsPage && m_settingsPage->isVisible()) {
        if (event->key() == Qt::Key_Back || isSettingsKey) {
            hideSettings();
            return true;
        }
        return false;
    }
    if (isSettingsKey) {
        showSettings();
        return true;
    }

    const bool guideVisible = m_guideOverlay && m_guideOverlay->isVisible();
    const bool infoVisible = m_infoOverlay && m_infoOverlay->isVisible();
    const bool isSelect = isRemoteSelectKey(event);

    if (isSelect) {
        if (guideVisible) {
            tuneChannelId(m_guideOverlay->selectedChannelId());
            hideOverlays();
        } else if (infoVisible) {
            hideOverlays();
        } else {
            showInfoOverlay();
        }
        return true;
    }

    switch (event->key()) {
    case Qt::Key_HomePage:
    case Qt::Key_Guide:
        if (guideVisible) {
            hideOverlays();
        } else {
            showGuideOverlay();
        }
        return true;
    case Qt::Key_Info:
        if (infoVisible) {
            hideOverlays();
        } else {
            showInfoOverlay();
        }
        return true;
    case Qt::Key_Back:
        // Unlike Escape, Back never leaves fullscreen or quits the client.
        hideOverlays();
        return true;
    case Qt::Key_PageUp:
    case Qt::Key_ChannelUp:
        if (guideVisible) {
            m_guideOverlay->pageChannels(-1);
        } else {
            tuneRelative(1);
        }
        return true;
    case Qt::Key_PageDown:
    case Qt::Key_ChannelDown:
        if (guideVisible) {
            m_guideOverlay->pageChannels(1);
        } else {
            tuneRelative(-1);
        }
        return true;
    default:
        return false;
    }
}

bool MainWindow::handleChannelNumberKey(QKeyEvent *event)
{
    const int key = event->key();
    if (key >= Qt::Key_0 && key <= Qt::Key_9
        && !(event->modifiers() & (Qt::ControlModifier | Qt::AltModifier | Qt::MetaModifier))) {
        appendChannelNumberDigit(key - Qt::Key_0);
        return true;
    }
    if (m_channelNumberBuffer.isEmpty()) {
        return false;
    }

    if (key == Qt::Key_Enter || key == Qt::Key_Return) {
        commitChannelNumber();
        return true;
    }
    if (key == Qt::Key_Backspace) {
        m_channelNumberBuffer.chop(1);
        if (m_channelNumberBuffer.isEmpty()) {
            cancelChannelNumber();
        } else {
            statusBar()->showMessage(QStringLiteral("Channel %1").arg(m_channelNumberBuffer));
            m_channelTransitionOverlay->updateNumberEntry(m_channelNumberBuffer);
            m_channelNumberTimer->start();
        }
        return true;
    }
    if (key == Qt::Key_Escape) {
        cancelChannelNumber();
        return true;
    }
    return false;
}

void MainWindow::appendChannelNumberDigit(int digit)
{
    if (m_channels.empty()) {
        statusBar()->showMessage(QStringLiteral("No channels are available."), 2500);
        return;
    }

    const int maximumDigits = maximumChannelNumberDigits();
    if (m_channelNumberBuffer.size() >= maximumDigits) {
        m_channelNumberBuffer.clear();
    }
    m_channelNumberBuffer.append(QString::number(digit));
    statusBar()->showMessage(QStringLiteral("Channel %1").arg(m_channelNumberBuffer));
    if (m_channelNumberBuffer.size() == 1) {
        m_channelTransitionOverlay->showNumberEntry(m_channelNumberBuffer);
    } else {
        m_channelTransitionOverlay->updateNumberEntry(m_channelNumberBuffer);
    }
    m_channelNumberTimer->start();
}

void MainWindow::commitChannelNumber()
{
    if (m_channelNumberBuffer.isEmpty()) {
        return;
    }

    m_channelNumberTimer->stop();
    m_channelTransitionOverlay->hideNumberEntry();
    bool validNumber = false;
    const int channelNumber = m_channelNumberBuffer.toInt(&validNumber);
    m_channelNumberBuffer.clear();
    if (!validNumber) {
        return;
    }

    for (int index = 0; index < static_cast<int>(m_channels.size()); ++index) {
        if (m_channels.at(index).number != channelNumber) {
            continue;
        }
        if ((m_guideOverlay && m_guideOverlay->isVisible()) || (m_infoOverlay && m_infoOverlay->isVisible())) {
            hideOverlays();
        }
        tuneChannel(index);
        return;
    }

    statusBar()->showMessage(QStringLiteral("Channel %1 is unavailable.").arg(channelNumber), 2500);
}

void MainWindow::cancelChannelNumber()
{
    m_channelNumberTimer->stop();
    m_channelNumberBuffer.clear();
    m_channelTransitionOverlay->hideNumberEntry();
    statusBar()->clearMessage();
}

int MainWindow::maximumChannelNumberDigits() const
{
    int maximumDigits = 1;
    for (const Channel &channel : m_channels) {
        maximumDigits = qMax(maximumDigits, QString::number(channel.number).size());
    }
    return maximumDigits;
}

void MainWindow::setStartupState(StartupState state, const QString &title, const QString &detail)
{
    m_startupState = state;

    if (state == StartupState::Playing) {
        m_stateView->hide();
        updateChannelBugVisibility();
        return;
    }

    const QString safeTitle = title.toHtmlEscaped();
    const QString safeDetail = detail.toHtmlEscaped();
    if (state == StartupState::Offline) {
        m_stateStack->setCurrentWidget(m_offlineView);
    } else {
        m_stateStack->setCurrentWidget(m_genericStateView);
        m_stateLabel->setAlignment(Qt::AlignCenter);
        m_stateLabel->setText(QStringLiteral("<div style=\"font-size: 26px; font-weight: 800;\">%1</div><div style=\"font-size: 18px; font-weight: 500; color: #94a3b8; margin-top: 18px;\">%2</div>")
                                  .arg(safeTitle, safeDetail));
    }
    if (m_guideOverlay && m_guideOverlay->isVisible()) {
        m_restoreStateViewAfterGuide = true;
        m_stateView->hide();
        m_guideOverlay->raise();
        m_guideOverlay->setFocus(Qt::OtherFocusReason);
        return;
    }
    m_stateView->show();
    m_stateView->raise();
    updateChannelBugVisibility();
}

void MainWindow::keyPressEvent(QKeyEvent *event)
{
    switch (event->key()) {
    case Qt::Key_F11:
        toggleFullscreen();
        event->accept();
        return;
    case Qt::Key_Enter:
    case Qt::Key_Return:
        if (event->modifiers() & Qt::AltModifier) {
            toggleFullscreen();
            event->accept();
            return;
        }
        if (m_guideOverlay->isVisible()) {
            tuneChannelId(m_guideOverlay->selectedChannelId());
            hideOverlays();
            event->accept();
            return;
        }
        QMainWindow::keyPressEvent(event);
        return;
    case Qt::Key_I:
        showInfoOverlay();
        event->accept();
        return;
    case Qt::Key_G:
        showGuideOverlay();
        event->accept();
        return;
    case Qt::Key_S:
        if (m_settingsPage->isVisible()) hideSettings(); else showSettings();
        event->accept();
        return;
    case Qt::Key_Escape:
        if ((m_guideOverlay && m_guideOverlay->isVisible())
            || (m_infoOverlay && m_infoOverlay->isVisible())) {
            hideOverlays();
        } else if (isFullScreen()) {
            showNormal();
            updateOverlayGeometry();
            updateChannelBugGeometry();
        } else {
            close();
        }
        event->accept();
        return;
    case Qt::Key_Up:
        if (m_guideOverlay->isVisible()) {
            m_guideOverlay->scrollChannels(-1);
            event->accept();
            return;
        }
        tuneRelative(1);
        event->accept();
        return;
    case Qt::Key_Down:
        if (m_guideOverlay->isVisible()) {
            m_guideOverlay->scrollChannels(1);
            event->accept();
            return;
        }
        tuneRelative(-1);
        event->accept();
        return;
    case Qt::Key_Right:
        if (m_guideOverlay->isVisible()) {
            m_guideOverlay->scrollTimeline(1);
            event->accept();
            return;
        }
        QMainWindow::keyPressEvent(event);
        return;
    case Qt::Key_Left:
        if (m_guideOverlay->isVisible()) {
            m_guideOverlay->scrollTimeline(-1);
            event->accept();
            return;
        }
        QMainWindow::keyPressEvent(event);
        return;
    default:
        // Logs remote buttons that have no binding yet, so they can be mapped.
        qWarning().nospace() << "RetroTV unhandled key: qt=0x" << Qt::hex << event->key()
                             << " scancode=" << Qt::dec << event->nativeScanCode()
                             << " text=" << event->text();
        QMainWindow::keyPressEvent(event);
    }
}

void MainWindow::changeEvent(QEvent *event)
{
    QMainWindow::changeEvent(event);
    if (event->type() != QEvent::WindowStateChange) {
        return;
    }

    // Fullscreen is the appliance mode: no status bar, no mouse pointer.
    // Windowed mode keeps both for development on a desktop.
    const bool fullScreen = isFullScreen();
    statusBar()->setVisible(!fullScreen);
    if (fullScreen && !m_cursorHidden) {
        QApplication::setOverrideCursor(Qt::BlankCursor);
        m_cursorHidden = true;
    } else if (!fullScreen && m_cursorHidden) {
        QApplication::restoreOverrideCursor();
        m_cursorHidden = false;
    }
}

void MainWindow::toggleFullscreen()
{
    if (isFullScreen()) {
        showNormal();
    } else {
        showFullScreen();
    }
    updateOverlayGeometry();
    updateChannelBugGeometry();
}

void MainWindow::moveEvent(QMoveEvent *event)
{
    QMainWindow::moveEvent(event);
    updateOverlayGeometry();
}

void MainWindow::resizeEvent(QResizeEvent *event)
{
    QMainWindow::resizeEvent(event);
    updateOverlayGeometry();
}

void MainWindow::tuneChannel(int channelIndex, bool showTransition)
{
    if (m_channels.empty()) {
        setStartupState(
            StartupState::NoChannels,
            QStringLiteral("No channels configured"),
            QStringLiteral("Create channels in the RetroTV backend, then restart or retune the client."));
        statusBar()->showMessage(QStringLiteral("No backend channels loaded."));
        return;
    }

    if (channelIndex < 0) {
        channelIndex = m_channels.size() - 1;
    } else if (channelIndex >= m_channels.size()) {
        channelIndex = 0;
    }

	m_currentChannelIndex = channelIndex;
	const int tuneGeneration = ++m_tuneGeneration;
	m_waitingForSchedule = false;

	const Channel &channel = m_channels.at(m_currentChannelIndex);
	m_guideOverlay->setCurrentChannelId(channel.id);
	const bool alreadyPlaying = m_startupState == StartupState::Playing && !m_stateView->isVisible();
	const bool useTransition = showTransition && alreadyPlaying;
	if (!alreadyPlaying) {
		setStartupState(
			StartupState::Tuning,
			QStringLiteral("Tuning channel %1").arg(channel.number),
			QString::fromStdString(channel.name));
	} else if (useTransition) {
		showChannelTransition(channel);
		qApp->processEvents();
	}
	loadChannelBugLogo(QString::fromStdString(channel.logoUrl), tuneGeneration);
	if (m_tuneReply) {
		QNetworkReply *staleReply = m_tuneReply;
		m_tuneReply = nullptr;
		staleReply->abort();
		staleReply->deleteLater();
	}
	const QUrl tuneURL(QStringLiteral("%1/api/channels/%2/tune").arg(m_backendURL).arg(channel.id));
	QNetworkRequest request(tuneURL);
	request.setHeader(QNetworkRequest::UserAgentHeader, QStringLiteral("retrotv-client/0.1"));
	request.setTransferTimeout(35000);
	QNetworkReply *reply = m_tuneNetworkManager->get(request);
	m_tuneReply = reply;
	connect(reply, &QNetworkReply::finished, this, [this, reply, tuneURL, channel, useTransition, tuneGeneration]() {
		if (m_tuneReply == reply) {
			m_tuneReply = nullptr;
		}
		const QNetworkReply::NetworkError networkError = reply->error();
		const int statusCode = reply->attribute(QNetworkRequest::HttpStatusCodeAttribute).toInt();
		const QByteArray body = reply->readAll();
		reply->deleteLater();
		if (tuneGeneration != m_tuneGeneration) {
			return;
		}
		if (networkError != QNetworkReply::NoError || statusCode >= 400) {
			qWarning() << "RetroTV tune failed" << tuneURL << reply->errorString() << statusCode << QString::fromUtf8(body).left(500);
			showMediaUnavailable(channel, QStringLiteral("The backend tune request failed."));
			return;
		}

		const std::optional<PlaybackTarget> target = playbackTargetFromResponse(tuneURL, body);
		if (target) {
			applyPlaybackTarget(channel, *target, useTransition);
			return;
		}

		showNoCurrentProgram(channel);
	});
}

void MainWindow::applyPlaybackTarget(const Channel &channel, const PlaybackTarget &target, bool useTransition)
{
	m_waitingForSchedule = false;
	m_nowPlayingInfo = NowPlayingInfo{
		channel.number,
		QString::fromStdString(channel.name),
		target.title,
		target.timeRange,
		target.description,
		target.posterPath,
		target.upNextTitle,
		target.upNextTimeRange,
		target.startTime,
		target.endTime,
		target.year,
		target.rating,
		target.genre,
		target.seasonNumber,
		target.episodeNumber,
		target.hasClosedCaptions,
		target.isHD,
	};
	m_infoOverlay->setNowPlaying(m_nowPlayingInfo);
	if (useTransition) {
		m_channelTransitionOverlay->updateTransitionInfo(ChannelTransitionInfo{
			channel.number,
			QString::fromStdString(channel.name),
			target.title,
			target.timeRange,
		});
	}
	m_playerWidget->loadMedia(target.streamURL, target.seekSeconds);
	statusBar()->showMessage(QStringLiteral("%1: %2 - %3")
								 .arg(channel.number)
								 .arg(QString::fromStdString(channel.name), target.title));
}

void MainWindow::showNoCurrentProgram(const Channel &channel)
{
	m_waitingForSchedule = true;
    if (m_channelTransitionOverlay) {
        m_channelTransitionOverlay->clearBlackoutWhenReady();
    }
    m_nowPlayingInfo = NowPlayingInfo{
        channel.number,
        QString::fromStdString(channel.name),
        QStringLiteral("No current schedule item"),
        QString(),
        QStringLiteral("The backend did not return a current programme for this channel."),
        QString(),
        QString(),
        QString(),
    };
    m_infoOverlay->setNowPlaying(m_nowPlayingInfo);
    setStartupState(
        StartupState::MediaUnavailable,
        QStringLiteral("No current program"),
        QStringLiteral("%1 did not return a scheduled item for right now.").arg(QString::fromStdString(channel.name)));
    statusBar()->showMessage(QStringLiteral("%1: %2 (no current schedule item)")
                                 .arg(channel.number)
                                 .arg(QString::fromStdString(channel.name)));

	const int retryGeneration = m_tuneGeneration;
	QTimer::singleShot(3000, this, [this, retryGeneration]() {
		if (!m_waitingForSchedule || retryGeneration != m_tuneGeneration || m_channels.empty()) {
			return;
		}
		tuneChannel(m_currentChannelIndex, false);
	});
}

void MainWindow::showMediaUnavailable(const Channel &channel, const QString &message)
{
	m_waitingForSchedule = false;
    m_nowPlayingInfo = NowPlayingInfo{
        channel.number,
        QString::fromStdString(channel.name),
        QStringLiteral("Media unavailable"),
        QString(),
        message,
        QString(),
        QString(),
        QString(),
    };
    m_infoOverlay->setNowPlaying(m_nowPlayingInfo);
    setStartupState(
        StartupState::MediaUnavailable,
        QStringLiteral("Media unavailable"),
        QStringLiteral("%1 could not play the scheduled media. %2")
            .arg(QString::fromStdString(channel.name), message));
    statusBar()->showMessage(QStringLiteral("%1: %2 (media unavailable)")
                                 .arg(channel.number)
                                 .arg(QString::fromStdString(channel.name)));
}

void MainWindow::tuneChannelId(int channelId)
{
    if (channelId <= 0) {
        return;
    }

    for (int index = 0; index < static_cast<int>(m_channels.size()); ++index) {
        if (m_channels.at(index).id == channelId) {
            tuneChannel(index);
            return;
        }
    }
}

void MainWindow::tuneRelative(int offset)
{
    tuneChannel(m_currentChannelIndex + offset);
}

void MainWindow::showChannelTransition(const Channel &channel, const QString &title, const QString &timeRange)
{
    if (!m_channelTransitionOverlay) {
        return;
    }

    m_channelTransitionOverlay->setGeometry(m_rootWidget->rect());
    m_channelTransitionOverlay->startTransition(ChannelTransitionInfo{
        channel.number,
        QString::fromStdString(channel.name),
        title,
        timeRange,
    });
}

void MainWindow::showInfoOverlay()
{
    updateOverlayGeometry();
    m_infoOverlay->setNowPlaying(m_nowPlayingInfo);
    m_guideOverlay->hide();
    m_infoOverlay->show();
    m_infoOverlay->raise();
    updateChannelBugVisibility();
    setFocus();
}

void MainWindow::showGuideOverlay()
{
    updateOverlayGeometry();
    if (!m_channels.empty()) {
        m_guideOverlay->setCurrentChannelId(m_channels.at(m_currentChannelIndex).id);
    }
    m_guideOverlay->resetTimelineToNow();
    m_restoreStateViewAfterGuide = m_stateView->isVisible();
    if (m_restoreStateViewAfterGuide) {
        m_stateView->hide();
    }
    m_infoOverlay->hide();
    m_guideOverlay->show();
    m_guideOverlay->raise();
    updateChannelBugVisibility();
    m_guideOverlay->setFocus(Qt::OtherFocusReason);
    refreshGuide();
}

void MainWindow::refreshGuide()
{
    if (m_guideRefreshInFlight || !m_backendConnected || m_channels.empty()) {
        return;
    }
    m_guideRefreshInFlight = true;

    const QString backendURL = m_backendURL;
    auto *watcher = new QFutureWatcher<XMLTVGuide>(this);
    connect(watcher, &QFutureWatcher<XMLTVGuide>::finished, this, [this, watcher, backendURL]() {
        watcher->deleteLater();
        m_guideRefreshInFlight = false;
        XMLTVGuide guide = watcher->result();
        // Keep the current guide if the backend changed meanwhile or the
        // fetch failed, rather than blanking it.
        if (backendURL != m_backendURL || guide.programmes.empty()) {
            return;
        }
        m_guideOverlay->setGuide(std::move(guide));
    });
    watcher->setFuture(QtConcurrent::run([backendURL, channels = m_channels]() {
        return loadGuide(backendURL, channels);
    }));
}

void MainWindow::hideOverlays()
{
    m_infoOverlay->hide();
    m_guideOverlay->hide();
    if (m_restoreStateViewAfterGuide && !m_stateView->isVisible() && m_startupState != StartupState::Playing) {
        m_stateView->show();
        m_stateView->raise();
    }
    m_restoreStateViewAfterGuide = false;
    updateChannelBugVisibility();
    setFocus();
}

void MainWindow::updateOverlayGeometry()
{
    if (!m_infoOverlay || !m_guideOverlay || !m_channelTransitionOverlay || !m_rootWidget) {
        return;
    }

    m_infoOverlay->setGeometry(m_rootWidget->rect());
    m_guideOverlay->setGeometry(m_rootWidget->rect());
    m_channelTransitionOverlay->setGeometry(m_rootWidget->rect());
	updateChannelBugGeometry();
}

void MainWindow::loadChannelBugLogo(const QString &logoURL, int tuneGeneration)
{
	if (!m_channelBugLabel) {
		return;
	}

	if (logoURL.isEmpty()) {
		m_channelBugPixmap = QPixmap();
		m_channelBugLabel->clear();
		m_channelBugLabel->hide();
		return;
	}

	m_channelBugLabel->clear();
	m_channelBugLabel->hide();
	m_channelBugLabel->update();

	auto *watcher = new QFutureWatcher<QPixmap>(this);
	connect(watcher, &QFutureWatcher<QPixmap>::finished, this, [this, watcher, tuneGeneration]() {
		watcher->deleteLater();
		if (tuneGeneration != m_tuneGeneration) {
			return;
		}

		const QPixmap pixmap = watcher->result();
		if (pixmap.isNull()) {
			m_channelBugPixmap = QPixmap();
			m_channelBugLabel->clear();
			m_channelBugLabel->hide();
			return;
		}

		m_channelBugPixmap = pixmap;
		updateChannelBugGeometry();
		updateChannelBugVisibility();
	});
	watcher->setFuture(QtConcurrent::run([logoURL]() -> QPixmap {
		if (logoURL.startsWith(QStringLiteral("http://")) || logoURL.startsWith(QStringLiteral("https://"))) {
			auto body = fetchText(QUrl(logoURL));
			if (!body) {
				return {};
			}

			QPixmap pixmap;
			const QByteArray imageData(body->data(), static_cast<qsizetype>(body->size()));
			pixmap.loadFromData(imageData);
			return pixmap;
		}

		return QPixmap(logoURL);
	}));
}

void MainWindow::updateChannelBugGeometry()
{
    if (!m_channelBugLabel || !m_rootWidget) {
        return;
    }

    const QSize rootSize = m_rootWidget->size();
    const int maxWidth = qMax(96, rootSize.width() / 9);
    const int maxHeight = qMax(54, rootSize.height() / 10);
    const QSize bugSize(maxWidth, maxHeight);
    const int margin = qMax(28, rootSize.width() / 48);
    m_channelBugLabel->setGeometry(
        rootSize.width() - bugSize.width() - margin,
        rootSize.height() - bugSize.height() - margin,
        bugSize.width(),
        bugSize.height());

    if (m_channelBugPixmap.isNull()) {
        m_channelBugLabel->clear();
        return;
    }

    QPixmap canvas(bugSize);
    canvas.fill(Qt::transparent);

    QSize logoSize = m_channelBugPixmap.size();
    logoSize.scale(bugSize, Qt::KeepAspectRatio);
    const QPixmap scaledLogo = m_channelBugPixmap.scaled(
        logoSize,
        Qt::KeepAspectRatio,
        Qt::SmoothTransformation);

    QPainter painter(&canvas);
    painter.setRenderHint(QPainter::SmoothPixmapTransform, true);
    painter.drawPixmap(
        (bugSize.width() - scaledLogo.width()) / 2,
        (bugSize.height() - scaledLogo.height()) / 2,
        scaledLogo);
    painter.end();

    m_channelBugLabel->setPixmap(canvas);
}

void MainWindow::updateChannelBugVisibility()
{
    if (!m_channelBugLabel) {
        return;
    }

    const bool overlayVisible =
        (m_infoOverlay && m_infoOverlay->isVisible()) ||
        (m_guideOverlay && m_guideOverlay->isVisible()) ||
        (m_stateView && m_stateView->isVisible());
    const bool shouldShow = !overlayVisible && !m_channelBugPixmap.isNull();
    m_channelBugLabel->setVisible(shouldShow);
    if (shouldShow) {
        m_channelBugLabel->raise();
    }
}
