#include "InfoOverlay.h"

#include <QFrame>
#include <QGridLayout>
#include <QHBoxLayout>
#include <QKeyEvent>
#include <QLabel>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QNetworkRequest>
#include <QPainter>
#include <QPixmap>
#include <QProgressBar>
#include <QSizePolicy>
#include <QTextLayout>
#include <QTextOption>
#include <QTime>
#include <QTimer>
#include <QUrl>
#include <QVector>
#include <QVBoxLayout>

class ThreeLineLabel final : public QLabel
{
public:
    explicit ThreeLineLabel(QWidget *parent = nullptr)
        : QLabel(parent)
    {
        setWordWrap(true);
        setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Fixed);
    }

    QSize sizeHint() const override
    {
        QSize result = QLabel::sizeHint();
        result.setHeight(fontMetrics().lineSpacing() * 3);
        return result;
    }

    QSize minimumSizeHint() const override
    {
        return QSize(0, fontMetrics().lineSpacing() * 3);
    }

protected:
    void paintEvent(QPaintEvent *) override
    {
        const QRectF bounds = contentsRect();
        if (bounds.width() <= 0 || text().isEmpty()) {
            return;
        }

        QTextLayout layout(text(), font());
        QTextOption option;
        option.setWrapMode(QTextOption::WordWrap);
        layout.setTextOption(option);

        QVector<QTextLine> lines;
        qreal y = 0;
        layout.beginLayout();
        for (int index = 0; index < 3; ++index) {
            QTextLine line = layout.createLine();
            if (!line.isValid()) {
                break;
            }
            line.setLineWidth(bounds.width());
            line.setPosition(QPointF(0, y));
            y += line.height();
            lines.push_back(line);
        }
        layout.endLayout();

        QPainter painter(this);
        painter.setFont(font());
        painter.setPen(palette().color(QPalette::WindowText));

        for (int index = 0; index < lines.size(); ++index) {
            const QTextLine &line = lines.at(index);
            const bool lastLine = index == lines.size() - 1;
            const int consumedCharacters = line.textStart() + line.textLength();
            const bool truncated = lastLine && consumedCharacters < text().size();

            if (truncated) {
                const QString remainingText = text().mid(line.textStart()).simplified();
                const QString elidedText = fontMetrics().elidedText(
                    remainingText, Qt::ElideRight, static_cast<int>(bounds.width()));
                const QRectF lineBounds(bounds.left(),
                                        bounds.top() + line.y(),
                                        bounds.width(),
                                        line.height());
                painter.drawText(lineBounds, Qt::AlignLeft | Qt::AlignVCenter, elidedText);
            } else {
                line.draw(&painter, bounds.topLeft());
            }
        }
    }
};

class MetadataIconLabel final : public QLabel
{
public:
    enum class Kind {
        Clock,
        Calendar,
        Category,
    };

    explicit MetadataIconLabel(Kind kind, QWidget *parent = nullptr)
        : QLabel(parent)
        , m_kind(kind)
    {
        setFixedSize(24, 28);
    }

protected:
    void paintEvent(QPaintEvent *) override
    {
        QPainter painter(this);
        painter.setRenderHint(QPainter::Antialiasing);
        painter.setPen(QPen(QColor(QStringLiteral("#4aa8ff")), 2));
        painter.setBrush(Qt::NoBrush);

        if (m_kind == Kind::Clock) {
            painter.drawEllipse(QRectF(3, 5, 18, 18));
            painter.drawLine(QPointF(12, 14), QPointF(12, 9));
            painter.drawLine(QPointF(12, 14), QPointF(16, 17));
            return;
        }

        if (m_kind == Kind::Calendar) {
            painter.drawRoundedRect(QRectF(3, 6, 18, 17), 2, 2);
            painter.drawLine(QPointF(3, 11), QPointF(21, 11));
            painter.drawLine(QPointF(8, 4), QPointF(8, 8));
            painter.drawLine(QPointF(16, 4), QPointF(16, 8));
            painter.drawPoint(QPointF(8, 15));
            painter.drawPoint(QPointF(12, 15));
            painter.drawPoint(QPointF(16, 15));
            painter.drawPoint(QPointF(8, 19));
            painter.drawPoint(QPointF(12, 19));
            return;
        }

        const QPolygonF tag({QPointF(3, 9), QPointF(10, 4), QPointF(21, 14),
                             QPointF(12, 23), QPointF(3, 16)});
        painter.drawPolygon(tag);
        painter.drawEllipse(QPointF(8, 10), 1.5, 1.5);
    }

private:
    Kind m_kind;
};

namespace {

QLabel *makeLabel(const QString &text, const QString &objectName = QString())
{
    auto *label = new QLabel(text);
    if (!objectName.isEmpty()) {
        label->setObjectName(objectName);
    }
    label->setWordWrap(true);
    return label;
}

QLabel *makeBadge(const QString &text)
{
    auto *badge = makeLabel(text, QStringLiteral("badge"));
    badge->setAlignment(Qt::AlignCenter);
    badge->setFixedHeight(42);
    return badge;
}

void retainSizeWhenHidden(QWidget *widget)
{
    QSizePolicy policy = widget->sizePolicy();
    policy.setRetainSizeWhenHidden(true);
    widget->setSizePolicy(policy);
}

void ignoreHorizontalTextSize(QLabel *label)
{
    QSizePolicy policy = label->sizePolicy();
    policy.setHorizontalPolicy(QSizePolicy::Ignored);
    label->setSizePolicy(policy);
}

}

InfoOverlay::InfoOverlay(QWidget *parent)
    : QWidget(parent)
{
	m_networkManager = new QNetworkAccessManager(this);
    setFocusPolicy(Qt::StrongFocus);

    setStyleSheet(QStringLiteral(R"(
        InfoOverlay {
            background: transparent;
        }
        QFrame#contentPanel {
            background: rgba(0, 0, 0, 218);
            border: 1px solid rgba(255, 255, 255, 52);
            border-radius: 16px;
        }
        QFrame#footerBar {
            background: rgba(0, 0, 0, 210);
        }
        QLabel {
            color: #f2f4f8;
            font-size: 22px;
        }
        QLabel#sectionLabel {
            color: #8fc6ff;
            font-size: 14px;
            letter-spacing: 0px;
            text-transform: uppercase;
        }
        QLabel#channelNumber {
            color: #ffffff;
            font-size: 46px;
            font-weight: 700;
        }
        QLabel#channelPrefix {
            color: #4aa8ff;
            font-size: 22px;
            font-weight: 600;
        }
        QLabel#networkName {
            color: #ffffff;
            font-size: 34px;
            font-weight: 700;
        }
        QLabel#episodeTitle {
            color: #4aa8ff;
            font-size: 30px;
            font-weight: 700;
        }
        QLabel#description {
            color: rgba(255, 255, 255, 190);
            font-size: 20px;
            line-height: 1.2;
        }
        QLabel#timeLabel {
            color: #ffffff;
            font-size: 26px;
        }
        QLabel#remainingLabel, QLabel#metadataIcon {
            color: #4aa8ff;
            font-size: 17px;
        }
        QLabel#metadataValue {
            color: rgba(255, 255, 255, 220);
            font-size: 18px;
        }
        QProgressBar#programProgress {
            min-height: 10px;
            max-height: 10px;
            border: none;
            border-radius: 5px;
            background: rgba(255, 255, 255, 35);
        }
        QProgressBar#programProgress::chunk {
            border-radius: 5px;
            background: #4aa8ff;
        }
        QLabel#poster {
            background: qlineargradient(x1:0, y1:0, x2:1, y2:1,
                                        stop:0 #113c86, stop:0.48 #07162c, stop:1 #d8eefc);
            border-radius: 6px;
            color: #ffe16a;
            font-size: 20px;
            font-weight: 700;
            padding: 12px;
        }
        QLabel#badge {
            border: 1px solid rgba(255, 255, 255, 80);
            border-radius: 5px;
            background: rgba(255, 255, 255, 18);
            color: #ffffff;
            font-size: 16px;
            padding: 4px 10px;
        }
        QLabel#footerAction {
            color: rgba(255, 255, 255, 210);
            font-size: 20px;
        }
        QLabel#footerIcon {
            border: 2px solid rgba(255, 255, 255, 180);
            border-radius: 18px;
            color: #ffffff;
            font-size: 17px;
            font-weight: 700;
            min-width: 36px;
            min-height: 36px;
            max-width: 36px;
            max-height: 36px;
        }
    )"));

    auto *rootLayout = new QVBoxLayout(this);
    rootLayout->setContentsMargins(28, 28, 28, 0);
    rootLayout->addStretch(1);

    auto *panel = new QFrame(this);
    panel->setObjectName(QStringLiteral("contentPanel"));
    panel->setFixedHeight(460);

    auto *panelLayout = new QHBoxLayout(panel);
    panelLayout->setContentsMargins(40, 24, 40, 24);
    panelLayout->setSpacing(40);

    auto *leftLayout = new QGridLayout();
    leftLayout->setHorizontalSpacing(24);
    leftLayout->setVerticalSpacing(8);

    auto *section = makeLabel(QStringLiteral("NOW PLAYING"), QStringLiteral("sectionLabel"));
    section->setFixedHeight(22);
    leftLayout->addWidget(section, 0, 0, 1, 2);

    auto *channelRow = new QHBoxLayout();
    channelRow->setSpacing(12);
    channelRow->addWidget(makeLabel(QStringLiteral("CH"), QStringLiteral("channelPrefix")));
    m_channelNumberLabel = makeLabel(QStringLiteral("001"), QStringLiteral("channelNumber"));
    m_channelNumberLabel->setFixedSize(108, 58);
    m_channelNumberLabel->setAlignment(Qt::AlignLeft | Qt::AlignVCenter);
    channelRow->addWidget(m_channelNumberLabel);
    channelRow->addSpacing(16);
    channelRow->addWidget(makeLabel(QStringLiteral("|")));
    channelRow->addSpacing(12);
    m_channelNameLabel = makeLabel(QStringLiteral("Retro TV"), QStringLiteral("networkName"));
    m_channelNameLabel->setWordWrap(false);
    m_channelNameLabel->setFixedSize(360, 58);
    m_channelNameLabel->setAlignment(Qt::AlignLeft | Qt::AlignVCenter);
    channelRow->addWidget(m_channelNameLabel);
    channelRow->addStretch(1);
    leftLayout->addLayout(channelRow, 1, 0, 1, 2);

    m_posterLabel = makeLabel(QStringLiteral("RETRO\nTV"), QStringLiteral("poster"));
    m_posterLabel->setAlignment(Qt::AlignCenter);
    m_posterLabel->setFixedSize(160, 210);
    m_posterLabel->setScaledContents(false);
    leftLayout->addWidget(m_posterLabel, 2, 0, 4, 1, Qt::AlignTop);

    auto *episodeLayout = new QVBoxLayout();
    episodeLayout->setSpacing(10);
    m_titleLabel = makeLabel(QStringLiteral("No programme loaded"), QStringLiteral("episodeTitle"));
    m_titleLabel->setWordWrap(false);
    m_titleLabel->setFixedHeight(42);
    ignoreHorizontalTextSize(m_titleLabel);
    m_timeLabel = makeLabel(QStringLiteral("Loading schedule"));
    m_timeLabel->setWordWrap(false);
    m_timeLabel->setFixedHeight(32);
    ignoreHorizontalTextSize(m_timeLabel);
    m_descriptionLabel = new ThreeLineLabel();
    m_descriptionLabel->setObjectName(QStringLiteral("description"));
    m_descriptionLabel->setText(QStringLiteral("Tune to a backend channel to see what is playing."));
    m_descriptionLabel->ensurePolished();
    const int descriptionHeight = qMax(96, m_descriptionLabel->fontMetrics().lineSpacing() * 3 + 12);
    m_descriptionLabel->setFixedHeight(descriptionHeight);
    episodeLayout->addWidget(m_titleLabel);
    episodeLayout->addWidget(m_timeLabel);
    episodeLayout->addWidget(m_descriptionLabel);

    auto *badgeRow = new QHBoxLayout();
    badgeRow->setSpacing(12);
    m_hdBadge = makeBadge(QStringLiteral("HD"));
    m_hdBadge->setFixedWidth(52);
    retainSizeWhenHidden(m_hdBadge);
    m_hdBadge->hide();
    badgeRow->addWidget(m_hdBadge);
    badgeRow->addWidget(makeBadge(QStringLiteral("STEREO")));
    m_ccBadge = makeBadge(QStringLiteral("CC"));
    m_ccBadge->setFixedWidth(52);
    retainSizeWhenHidden(m_ccBadge);
    m_ccBadge->hide();
    badgeRow->addWidget(m_ccBadge);
    badgeRow->addWidget(makeBadge(QStringLiteral("TV-PG")));
    badgeRow->addStretch(1);
    episodeLayout->addLayout(badgeRow);

    leftLayout->addLayout(episodeLayout, 2, 1, 4, 1);
    panelLayout->addLayout(leftLayout, 3);

    auto *divider = new QFrame(panel);
    divider->setFrameShape(QFrame::VLine);
    divider->setStyleSheet(QStringLiteral("color: rgba(255, 255, 255, 70);"));
    panelLayout->addWidget(divider);

    auto *rightLayout = new QVBoxLayout();
    rightLayout->setSpacing(12);
    m_clockLabel = makeLabel(QTime::currentTime().toString(QStringLiteral("h:mm AP")), QStringLiteral("timeLabel"));
    m_clockLabel->setFixedHeight(36);
    m_clockLabel->setWordWrap(false);
    rightLayout->addWidget(m_clockLabel, 0, Qt::AlignRight);
    rightLayout->addSpacing(8);

    auto *remainingRow = new QHBoxLayout();
    remainingRow->setSpacing(10);
    remainingRow->addWidget(new MetadataIconLabel(MetadataIconLabel::Kind::Clock));
    m_remainingLabel = makeLabel(QStringLiteral("0 MIN REMAINING"), QStringLiteral("remainingLabel"));
    m_remainingLabel->setFixedHeight(28);
    m_remainingLabel->setWordWrap(false);
    ignoreHorizontalTextSize(m_remainingLabel);
    remainingRow->addWidget(m_remainingLabel);
    remainingRow->addStretch(1);
    rightLayout->addLayout(remainingRow);

    m_progressBar = new QProgressBar();
    m_progressBar->setObjectName(QStringLiteral("programProgress"));
    m_progressBar->setRange(0, 1000);
    m_progressBar->setTextVisible(false);
    rightLayout->addWidget(m_progressBar);

    auto *yearRow = new QHBoxLayout();
    yearRow->setSpacing(10);
    m_yearIconLabel = new MetadataIconLabel(MetadataIconLabel::Kind::Calendar);
    retainSizeWhenHidden(m_yearIconLabel);
    yearRow->addWidget(m_yearIconLabel);
    m_yearRatingLabel = makeLabel(QString(), QStringLiteral("metadataValue"));
    m_yearRatingLabel->setFixedHeight(28);
    m_yearRatingLabel->setWordWrap(false);
    ignoreHorizontalTextSize(m_yearRatingLabel);
    retainSizeWhenHidden(m_yearRatingLabel);
    yearRow->addWidget(m_yearRatingLabel);
    yearRow->addStretch(1);
    rightLayout->addLayout(yearRow);

    auto *genreRow = new QHBoxLayout();
    genreRow->setSpacing(10);
    m_genreIconLabel = new MetadataIconLabel(MetadataIconLabel::Kind::Category);
    retainSizeWhenHidden(m_genreIconLabel);
    genreRow->addWidget(m_genreIconLabel);
    m_genreEpisodeLabel = makeLabel(QString(), QStringLiteral("metadataValue"));
    m_genreEpisodeLabel->setFixedHeight(28);
    m_genreEpisodeLabel->setWordWrap(false);
    ignoreHorizontalTextSize(m_genreEpisodeLabel);
    retainSizeWhenHidden(m_genreEpisodeLabel);
    genreRow->addWidget(m_genreEpisodeLabel);
    genreRow->addStretch(1);
    rightLayout->addLayout(genreRow);
    rightLayout->addStretch(1);

    auto *rightDivider = new QFrame(panel);
    rightDivider->setFrameShape(QFrame::HLine);
    rightDivider->setStyleSheet(QStringLiteral("color: rgba(255, 255, 255, 70);"));
    rightLayout->addWidget(rightDivider);
    rightLayout->addSpacing(8);
    auto *upNextSection = makeLabel(QStringLiteral("UP NEXT"), QStringLiteral("sectionLabel"));
    upNextSection->setFixedHeight(22);
    rightLayout->addWidget(upNextSection);
    m_upNextTitleLabel = makeLabel(QStringLiteral("No upcoming programme"));
    m_upNextTimeLabel = makeLabel(QString());
    m_upNextTitleLabel->setFixedHeight(56);
    m_upNextTimeLabel->setFixedHeight(30);
    m_upNextTimeLabel->setWordWrap(false);
    ignoreHorizontalTextSize(m_upNextTitleLabel);
    ignoreHorizontalTextSize(m_upNextTimeLabel);
    rightLayout->addWidget(m_upNextTitleLabel);
    rightLayout->addWidget(m_upNextTimeLabel);
    panelLayout->addLayout(rightLayout, 2);

    rootLayout->addWidget(panel);

    auto *footer = new QFrame(this);
    footer->setObjectName(QStringLiteral("footerBar"));
    footer->setFixedHeight(82);
    auto *footerLayout = new QHBoxLayout(footer);
    footerLayout->setContentsMargins(24, 20, 24, 20);
    footerLayout->setSpacing(54);

    const auto addFooterAction = [footerLayout](const QString &icon, const QString &text) {
        auto *row = new QHBoxLayout();
        row->setSpacing(14);
        auto *iconLabel = makeLabel(icon, QStringLiteral("footerIcon"));
        iconLabel->setAlignment(Qt::AlignCenter);
        row->addWidget(iconLabel);
        row->addWidget(makeLabel(text, QStringLiteral("footerAction")));
        footerLayout->addLayout(row);
    };

    addFooterAction(QStringLiteral("i"), QStringLiteral("More Info"));
    addFooterAction(QStringLiteral("OK"), QStringLiteral("Channel List"));
    addFooterAction(QStringLiteral("G"), QStringLiteral("TV Guide"));
    addFooterAction(QStringLiteral("Esc"), QStringLiteral("Back to TV"));
    footerLayout->addStretch(1);

    rootLayout->addWidget(footer);

    m_progressTimer = new QTimer(this);
    m_progressTimer->setInterval(1000);
    connect(m_progressTimer, &QTimer::timeout, this, &InfoOverlay::updateProgress);
    m_progressTimer->start();
}

void InfoOverlay::setNowPlaying(const NowPlayingInfo &info)
{
    m_channelNumberLabel->setText(info.channelNumber > 0
                                      ? QStringLiteral("%1").arg(info.channelNumber, 3, 10, QLatin1Char('0'))
                                      : QStringLiteral("---"));
    m_channelNameLabel->setText(info.channelName.isEmpty() ? QStringLiteral("Retro TV") : info.channelName);
    m_titleLabel->setText(info.title.isEmpty() ? QStringLiteral("No current programme") : info.title);
    m_timeLabel->setText(info.timeRange);
    m_descriptionLabel->setText(info.description.isEmpty()
                                    ? QStringLiteral("No description available.")
                                    : info.description);
    m_clockLabel->setText(QTime::currentTime().toString(QStringLiteral("h:mm AP")));
    m_upNextTitleLabel->setText(info.upNextTitle.isEmpty() ? QStringLiteral("No upcoming programme") : info.upNextTitle);
    m_upNextTimeLabel->setText(info.upNextTimeRange);
    m_programStart = info.startTime;
    m_programEnd = info.endTime;

    QStringList yearRating;
    if (info.year > 0) {
        yearRating.append(QString::number(info.year));
    }
    if (!info.rating.isEmpty()) {
        yearRating.append(info.rating);
    }
    m_yearRatingLabel->setText(yearRating.join(QStringLiteral("   |   ")));
    m_yearRatingLabel->setVisible(!yearRating.isEmpty());
    m_yearIconLabel->setVisible(!yearRating.isEmpty());

    QStringList genreEpisode;
    if (!info.genre.isEmpty()) {
        genreEpisode.append(info.genre);
    }
    if (info.seasonNumber > 0 && info.episodeNumber > 0) {
        genreEpisode.append(QStringLiteral("Season %1  •  Episode %2")
                                .arg(info.seasonNumber)
                                .arg(info.episodeNumber));
    }
    m_genreEpisodeLabel->setText(genreEpisode.join(QStringLiteral("   |   ")));
    m_genreEpisodeLabel->setVisible(!genreEpisode.isEmpty());
    m_genreIconLabel->setVisible(!genreEpisode.isEmpty());
    m_ccBadge->setVisible(info.hasClosedCaptions);
    m_hdBadge->setVisible(info.isHD);
    updateProgress();
    loadPoster(info.posterPath, info.title);
}

void InfoOverlay::updateProgress()
{
    const QDateTime now = QDateTime::currentDateTimeUtc();
    m_clockLabel->setText(QTime::currentTime().toString(QStringLiteral("h:mm AP")));
    const qint64 duration = m_programStart.secsTo(m_programEnd);
    if (!m_programStart.isValid() || !m_programEnd.isValid() || duration <= 0) {
        m_remainingLabel->setText(QStringLiteral("TIME UNAVAILABLE"));
        m_progressBar->setValue(0);
        return;
    }

    const qint64 elapsed = qBound<qint64>(0, m_programStart.secsTo(now), duration);
    const qint64 remainingSeconds = qMax<qint64>(0, now.secsTo(m_programEnd));
    const qint64 remainingMinutes = (remainingSeconds + 59) / 60;
    m_remainingLabel->setText(QStringLiteral("%1 MIN REMAINING").arg(remainingMinutes));
    m_progressBar->setValue(static_cast<int>((elapsed * 1000) / duration));
}

void InfoOverlay::loadPoster(const QString &path, const QString &fallback)
{
    m_posterLabel->setText(fallback.left(24).toUpper());
    m_posterLabel->setPixmap(QPixmap());
    if (m_posterReply) {
        QNetworkReply *staleReply = m_posterReply;
        m_posterReply = nullptr;
        staleReply->abort();
        staleReply->deleteLater();
    }
    if (path.isEmpty()) {
        return;
    }
    if (!path.startsWith(QStringLiteral("http://")) && !path.startsWith(QStringLiteral("https://"))) {
        const QPixmap pixmap(path);
        if (!pixmap.isNull()) {
            m_posterLabel->setText(QString());
            m_posterLabel->setPixmap(pixmap.scaled(m_posterLabel->size(), Qt::KeepAspectRatio, Qt::SmoothTransformation));
        }
        return;
    }

    QNetworkRequest request{QUrl(path)};
    request.setHeader(QNetworkRequest::UserAgentHeader, QStringLiteral("retrotv-client/0.1"));
    request.setTransferTimeout(3000);
    QNetworkReply *reply = m_networkManager->get(request);
    m_posterReply = reply;
    connect(reply, &QNetworkReply::finished, this, [this, reply]() {
        if (m_posterReply != reply) {
            reply->deleteLater();
            return;
        }
        m_posterReply = nullptr;
        const QByteArray body = reply->readAll();
        const bool succeeded = reply->error() == QNetworkReply::NoError;
        reply->deleteLater();
        QPixmap pixmap;
        if (succeeded && pixmap.loadFromData(body)) {
            m_posterLabel->setText(QString());
            m_posterLabel->setPixmap(pixmap.scaled(m_posterLabel->size(), Qt::KeepAspectRatio, Qt::SmoothTransformation));
        }
    });
}

void InfoOverlay::keyPressEvent(QKeyEvent *event)
{
    if (event->key() == Qt::Key_Escape) {
        emit dismissRequested();
        event->accept();
        return;
    }

    QWidget::keyPressEvent(event);
}
