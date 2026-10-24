#include "GuideOverlay.h"

#include <algorithm>
#include <limits>

#include <QBuffer>
#include <QCache>
#include <QDate>
#include <QDateTime>
#include <QFrame>
#include <QGridLayout>
#include <QHBoxLayout>
#include <QCoreApplication>
#include <QGuiApplication>
#include <QImageReader>
#include <QKeyEvent>
#include <QLabel>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QNetworkRequest>
#include <QPixmap>
#include <QPointer>
#include <QRegularExpression>
#include <QScrollArea>
#include <QScrollBar>
#include <QScreen>
#include <QStyle>
#include <QString>
#include <QTime>
#include <QTimeZone>
#include <QUrl>
#include <QVBoxLayout>

namespace {

constexpr int kMinimumVisibleRows = 4;
constexpr int kFallbackVisibleRows = 10;
constexpr int kVisibleHalfHourSlots = 6;

// The layout and stylesheet below are written for the default text size (the
// programme cell font size in px); other sizes scale everything from there.
struct GuideMetrics {
    int rowHeight;
    int channelColumnWidth;
    int channelNumberWidth;
    int channelLogoSize;
    int headerHeight;
    int dividerHeight;
    int timeHeaderHeight;
    int detailHeight;
    int footerHeight;
    int footerIconSize;
};

GuideMetrics guideMetrics(int textSize)
{
    const int delta = textSize - GuideOverlay::DefaultTextSize;
    return GuideMetrics{
        90 + 2 * delta,
        330 + 6 * delta,
        50 + 2 * delta,
        56 + delta,
        80 + (7 * delta) / 5,
        54 + delta,
        40 + delta,
        280 + 7 * delta,
        60 + delta,
        34 + (3 * delta) / 5,
    };
}
constexpr int kHalfHourSeconds = 30 * 60;
constexpr int kImageCacheSizeKiB = 64 * 1024;

QLabel *label(const QString &text, const QString &objectName = QString())
{
    auto *result = new QLabel(text);
    if (!objectName.isEmpty()) {
        result->setObjectName(objectName);
    }
    result->setWordWrap(true);
    return result;
}

void clearLayout(QLayout *layout)
{
    while (QLayoutItem *item = layout->takeAt(0)) {
        if (QWidget *widget = item->widget()) {
            delete widget;
        }
        if (QLayout *childLayout = item->layout()) {
            clearLayout(childLayout);
            delete childLayout;
        }
        delete item;
    }
}

QDateTime parseXmltvTime(const std::string &value)
{
    QString text = QString::fromStdString(value);
    if (text.size() < 14) {
        return {};
    }

    const QDateTime base = QDateTime::fromString(text.left(14), QStringLiteral("yyyyMMddhhmmss"));
    if (!base.isValid()) {
        return {};
    }

    const QString offset = text.mid(14).trimmed();
    if (offset.size() == 5 && (offset.front() == QLatin1Char('+') || offset.front() == QLatin1Char('-'))) {
        bool ok = false;
        const int hours = offset.mid(1, 2).toInt(&ok);
        if (ok) {
            const int minutes = offset.mid(3, 2).toInt(&ok);
            if (ok) {
                const int sign = offset.front() == QLatin1Char('-') ? -1 : 1;
                const int offsetSeconds = sign * ((hours * 60) + minutes) * 60;
                return QDateTime(base.date(), base.time(), QTimeZone::UTC).addSecs(-offsetSeconds).toLocalTime();
            }
        }
    }

    return base;
}

QString formatTime(const QDateTime &value)
{
    if (!value.isValid()) {
        return {};
    }
    return value.time().toString(QStringLiteral("h:mm AP"));
}

QString guideCellTitle(const XMLTVProgramme &programme)
{
    QString title = QString::fromStdString(programme.title).trimmed();
    static const QRegularExpression episodeSuffix(
        QStringLiteral("\\s+S\\d{2}E\\d{2}(?:\\s+-\\s+.*)?$"));
    title.remove(episodeSuffix);
    return title.trimmed();
}

QString programCellStyle(bool selected, bool currentlyAiring)
{
    if (selected) {
        return QStringLiteral("selectedProgramCell");
    }
    return currentlyAiring ? QStringLiteral("airingProgramCell") : QStringLiteral("programCell");
}

QCache<QString, QPixmap> &imageCache()
{
    static QCache<QString, QPixmap> cache(kImageCacheSizeKiB);
    return cache;
}

QSize imageTargetSize(const QLabel *label, qreal devicePixelRatio)
{
    return QSize(
        qMax(1, qRound(label->width() * devicePixelRatio)),
        qMax(1, qRound(label->height() * devicePixelRatio)));
}

QString imageCacheKey(const QString &path, const QSize &targetSize, qreal devicePixelRatio)
{
    return QStringLiteral("%1|%2x%3@%4")
        .arg(path)
        .arg(targetSize.width())
        .arg(targetSize.height())
        .arg(devicePixelRatio, 0, 'f', 2);
}

QPixmap decodePixmap(QIODevice *device, const QSize &targetSize, qreal devicePixelRatio)
{
    QImageReader reader(device);
    reader.setAutoTransform(true);
    const QSize sourceSize = reader.size();
    if (sourceSize.width() > targetSize.width() || sourceSize.height() > targetSize.height()) {
        reader.setScaledSize(sourceSize.scaled(targetSize, Qt::KeepAspectRatio));
    }

    QImage image = reader.read();
    if (image.isNull()) {
        return {};
    }
    if (image.width() > targetSize.width() || image.height() > targetSize.height()) {
        image = image.scaled(targetSize, Qt::KeepAspectRatio, Qt::SmoothTransformation);
    }

    QPixmap pixmap = QPixmap::fromImage(std::move(image));
    pixmap.setDevicePixelRatio(devicePixelRatio);
    return pixmap;
}

QPixmap decodePixmap(const QString &path, const QSize &targetSize, qreal devicePixelRatio)
{
    QImageReader reader(path);
    reader.setAutoTransform(true);
    const QSize sourceSize = reader.size();
    if (sourceSize.width() > targetSize.width() || sourceSize.height() > targetSize.height()) {
        reader.setScaledSize(sourceSize.scaled(targetSize, Qt::KeepAspectRatio));
    }

    QImage image = reader.read();
    if (image.isNull()) {
        return {};
    }
    if (image.width() > targetSize.width() || image.height() > targetSize.height()) {
        image = image.scaled(targetSize, Qt::KeepAspectRatio, Qt::SmoothTransformation);
    }

    QPixmap pixmap = QPixmap::fromImage(std::move(image));
    pixmap.setDevicePixelRatio(devicePixelRatio);
    return pixmap;
}

void cachePixmap(const QString &key, const QPixmap &pixmap)
{
    if (pixmap.isNull()) {
        return;
    }
    const qint64 bytes = static_cast<qint64>(pixmap.width()) * pixmap.height() * 4;
    const int costKiB = static_cast<int>(qMin<qint64>(
        std::numeric_limits<int>::max(), qMax<qint64>(1, (bytes + 1023) / 1024)));
    imageCache().insert(key, new QPixmap(pixmap), costKiB);
}

void setPixmapFromSource(QLabel *label, const std::string &source)
{
    const QString path = QString::fromStdString(source);
    const qreal devicePixelRatio = label->devicePixelRatioF();
    const QSize targetSize = imageTargetSize(label, devicePixelRatio);
    const QString cacheKey = imageCacheKey(path, targetSize, devicePixelRatio);
    if (const QPixmap *cached = imageCache().object(cacheKey)) {
        label->setText(QString());
        label->setPixmap(*cached);
        return;
    }

    const QUrl url(path);
    if (url.isValid() && (url.scheme() == QStringLiteral("http") || url.scheme() == QStringLiteral("https"))) {
        static auto *manager = new QNetworkAccessManager(QCoreApplication::instance());
        QNetworkRequest request(url);
        request.setTransferTimeout(1000);
        QNetworkReply *reply = manager->get(request);
        const QPointer<QLabel> guardedLabel(label);
        QObject::connect(reply, &QNetworkReply::finished, manager, [reply, guardedLabel, cacheKey, targetSize, devicePixelRatio]() {
            QPixmap pixmap;
            if (reply->error() == QNetworkReply::NoError) {
                QByteArray imageData = reply->readAll();
                QBuffer buffer(&imageData);
                buffer.open(QIODevice::ReadOnly);
                pixmap = decodePixmap(&buffer, targetSize, devicePixelRatio);
            }
            reply->deleteLater();
            cachePixmap(cacheKey, pixmap);
            if (guardedLabel && !pixmap.isNull()) {
                guardedLabel->setText(QString());
                guardedLabel->setPixmap(pixmap);
            }
        });
        return;
    }

    const QPixmap pixmap = decodePixmap(path, targetSize, devicePixelRatio);
    cachePixmap(cacheKey, pixmap);
    if (!pixmap.isNull()) {
        label->setText(QString());
        label->setPixmap(pixmap);
    }
}

void updatePosterLabel(QLabel *poster, const XMLTVProgramme *programme);

QLabel *posterLabel(const XMLTVProgramme *programme)
{
    auto *poster = label(QString(), QStringLiteral("poster"));
    poster->setAlignment(Qt::AlignCenter);
    poster->setFixedSize(112, 168);
    poster->setScaledContents(false);
    updatePosterLabel(poster, programme);
    return poster;
}

void updatePosterLabel(QLabel *poster, const XMLTVProgramme *programme)
{
    if (!poster) {
        return;
    }
    poster->setPixmap(QPixmap());
    poster->setText(programme ? QString::fromStdString(programme->title).left(24).toUpper() : QString());
    if (!programme || programme->icon.empty()) {
        return;
    }

    setPixmapFromSource(poster, programme->icon);
}

QWidget *channelCellWidget(const XMLTVChannel &channel, int channelNumber, const QString &objectName, const GuideMetrics &metrics)
{
    auto *cell = new QFrame();
    cell->setObjectName(objectName);
    cell->setFixedHeight(metrics.rowHeight);

    auto *layout = new QHBoxLayout(cell);
    layout->setContentsMargins(18, 0, 12, 0);
    layout->setSpacing(12);

    auto *number = label(QString::number(channelNumber), QStringLiteral("channelNumber"));
    number->setFixedWidth(metrics.channelNumberWidth);
    number->setAlignment(Qt::AlignLeft | Qt::AlignVCenter);
    layout->addWidget(number);

    if (!channel.icon.empty()) {
        auto *logo = label(QString(), QStringLiteral("channelLogo"));
        logo->setFixedSize(metrics.channelLogoSize, metrics.channelLogoSize);
        logo->setAlignment(Qt::AlignCenter);
        setPixmapFromSource(logo, channel.icon);
        layout->addWidget(logo);
    }

    auto *name = label(QString::fromStdString(channel.displayName), QStringLiteral("channelName"));
    name->setAlignment(Qt::AlignLeft | Qt::AlignVCenter);
    layout->addWidget(name, 1);
    return cell;
}

QString guideStyleSheet(int textSize)
{
    static const QString base = QStringLiteral(R"(
        GuideOverlay { background: #03080c; }
        QScrollArea#guideScrollArea, QScrollArea#guideScrollArea > QWidget, QScrollArea#guideScrollArea > QWidget > QWidget {
            background: #03080c;
            border: 0;
        }
        QLabel { color: #f5f6f8; font-size: 29px; }
        QLabel#brand { color: #f5f6f8; font-size: 48px; font-weight: 800; }
        QLabel#brandRed { color: #ff1f2d; font-size: 48px; font-weight: 800; }
        QLabel#screenTitle { color: #ffffff; font-size: 38px; font-weight: 700; }
        QLabel#clock { color: #ffffff; font-size: 38px; }
        QLabel#date { color: rgba(255, 255, 255, 170); font-size: 29px; }
        QLabel#axisLabel { color: rgba(255, 255, 255, 190); font-size: 28px; letter-spacing: 0px; }
        QFrame#channelCell, QFrame#selectedChannelCell {
            background: #071015;
            border-left: 1px solid #263138;
            border-top: 1px solid #263138;
        }
        QFrame#selectedChannelCell {
            background: #10181d;
            border: 1px solid #d01824;
        }
        QLabel#channelNumber, QLabel#channelName {
            color: #d4d9de;
            font-size: 29px;
        }
        QLabel#channelLogo {
            border: 0;
            background: transparent;
        }
        QLabel#programCell, QLabel#airingProgramCell, QLabel#selectedProgramCell {
            background: #10181d;
            border-left: 1px solid #2b353c;
            border-top: 1px solid #2b353c;
            color: #ffffff;
            font-size: 28px;
            padding: 8px 16px;
        }
        QLabel#airingProgramCell {
            background: #18262d;
        }
        QLabel#selectedProgramCell {
            background: #6f181f;
            border: 1px solid #d01824;
        }
        QFrame#gridFrame {
            border: 1px solid #263138;
            border-radius: 8px;
            background: #071015;
        }
        QFrame#detailPanel {
            border: 1px solid #263138;
            border-radius: 8px;
            background: #070e12;
        }
        QLabel#poster {
            border-radius: 6px;
            border: 1px solid #2f3a42;
            background: qlineargradient(x1:0, y1:0, x2:1, y2:1,
                                        stop:0 #053565, stop:0.45 #147eb5, stop:1 #0c1524);
            color: #ff2734;
            font-size: 28px;
            font-weight: 800;
            padding: 0;
        }
        QLabel#detailTitle { color: #ffffff; font-size: 40px; font-weight: 700; }
        QLabel#detailMeta { color: rgba(255, 255, 255, 185); font-size: 29px; }
        QLabel#description { color: rgba(255, 255, 255, 172); font-size: 30px; }
        QLabel#badge {
            border: 1px solid rgba(255, 255, 255, 70);
            border-radius: 4px;
            background: rgba(255, 255, 255, 16);
            font-size: 25px;
            padding: 4px 8px;
        }
        QLabel#footerText { color: rgba(255, 255, 255, 170); font-size: 26px; }
        QLabel#footerIcon, QLabel#yellowIcon, QLabel#blueIcon, QLabel#redIcon {
            border: 2px solid rgba(255, 255, 255, 190);
            border-radius: %ICON_RADIUS%px;
            color: #ffffff;
            font-size: 22px;
            min-width: %ICON_SIZE%px;
            min-height: %ICON_SIZE%px;
            max-width: %ICON_SIZE%px;
            max-height: %ICON_SIZE%px;
        }
        QLabel#yellowIcon { border-color: #ffd000; background: #ffd000; color: #111111; }
        QLabel#blueIcon { border-color: #1aa6e8; background: #1aa6e8; color: #051018; }
        QLabel#redIcon { border-color: #ff2b23; background: #ff2b23; color: #ffffff; }
    )");
    static const QRegularExpression fontSize(QStringLiteral("font-size: (\\d+)px"));

    const int delta = textSize - GuideOverlay::DefaultTextSize;
    const GuideMetrics metrics = guideMetrics(textSize);
    QString sheet;
    qsizetype last = 0;
    for (QRegularExpressionMatchIterator it = fontSize.globalMatch(base); it.hasNext();) {
        const QRegularExpressionMatch match = it.next();
        sheet += base.mid(last, match.capturedStart() - last);
        sheet += QStringLiteral("font-size: %1px").arg(std::max(8, match.captured(1).toInt() + delta));
        last = match.capturedEnd();
    }
    sheet += base.mid(last);
    sheet.replace(QStringLiteral("%ICON_SIZE%"), QString::number(metrics.footerIconSize));
    sheet.replace(QStringLiteral("%ICON_RADIUS%"), QString::number(metrics.footerIconSize / 2));
    return sheet;
}

int minutesBetween(const QDateTime &start, const QDateTime &stop)
{
    return static_cast<int>(start.secsTo(stop) / 60);
}

int halfHourSpan(const QDateTime &start, const QDateTime &stop)
{
    return std::max(1, (minutesBetween(start, stop) + 29) / 30);
}

void addFooterAction(QHBoxLayout *layout, const QString &icon, const QString &text, const QString &colorName = QString())
{
    auto *row = new QHBoxLayout();
    row->setSpacing(12);

    auto *iconLabel = label(icon, colorName.isEmpty() ? QStringLiteral("footerIcon") : colorName);
    iconLabel->setAlignment(Qt::AlignCenter);
    row->addWidget(iconLabel);
    row->addWidget(label(text, QStringLiteral("footerText")));

    layout->addLayout(row);
}

QDateTime halfHourFloor(const QDateTime &value)
{
    if (!value.isValid()) {
        return {};
    }

    const QTime time = value.time();
    const int minute = time.minute() >= 30 ? 30 : 0;
    return QDateTime(value.date(), QTime(time.hour(), minute));
}

QDateTime guideStartTime()
{
    const QDateTime now = QDateTime::currentDateTime();
    QDateTime start = halfHourFloor(now);
    if (!start.isValid()) {
        start = QDateTime(QDate::currentDate(), QTime(18, 0));
    }
    return start;
}

int channelIndexById(const XMLTVGuide &guide, int channelId)
{
    if (channelId <= 0) {
        return -1;
    }
    const std::string id = QStringLiteral("retrotv-%1").arg(channelId).toStdString();
    for (int index = 0; index < static_cast<int>(guide.channels.size()); ++index) {
        if (guide.channels.at(index).id == id) {
            return index;
        }
    }
    return -1;
}

}

GuideOverlay::GuideOverlay(QWidget *parent)
    : QWidget(parent)
{
    setAttribute(Qt::WA_StyledBackground);
    setAttribute(Qt::WA_ShowWithoutActivating);
    setFocusPolicy(Qt::StrongFocus);

    setStyleSheet(guideStyleSheet(m_textSize));

    // MainWindow calls setGuide() immediately after construction. Building once
    // with real guide data avoids deleting and recreating the top-level layout
    // during startup.
}

void GuideOverlay::prepareGuide()
{
    m_programmesByChannel.clear();
    m_guideStop = {};

    for (const XMLTVProgramme &programme : m_guide.programmes) {
        const QDateTime start = parseXmltvTime(programme.start);
        const QDateTime stop = parseXmltvTime(programme.stop);
        if (!start.isValid() || !stop.isValid()) {
            continue;
        }
        m_programmesByChannel[QString::fromStdString(programme.channelId)].push_back({&programme, start, stop});
        if (!m_guideStop.isValid() || stop > m_guideStop) {
            m_guideStop = stop;
        }
    }

    for (auto iterator = m_programmesByChannel.begin(); iterator != m_programmesByChannel.end(); ++iterator) {
        std::ranges::sort(iterator.value(), {}, &PreparedProgramme::start);
    }
}

const std::vector<GuideOverlay::PreparedProgramme> *GuideOverlay::programmesForChannel(
    const std::string &channelId) const
{
    const auto iterator = m_programmesByChannel.constFind(QString::fromStdString(channelId));
    return iterator == m_programmesByChannel.cend() ? nullptr : &iterator.value();
}

const GuideOverlay::PreparedProgramme *GuideOverlay::programmeAtTime(
    int channelId,
    const QDateTime &selectedTime) const
{
    const QString channelKey = channelId > 0 ? QStringLiteral("retrotv-%1").arg(channelId) : QString();
    const auto iterator = m_programmesByChannel.constFind(channelKey);
    const std::vector<PreparedProgramme> *programmes =
        iterator == m_programmesByChannel.cend() ? nullptr : &iterator.value();

    const auto exactProgramme = [](const std::vector<PreparedProgramme> *items, const QDateTime &time) {
        if (!items || !time.isValid()) {
            return static_cast<const PreparedProgramme *>(nullptr);
        }
        const auto candidate = std::upper_bound(
            items->begin(), items->end(), time,
            [](const QDateTime &value, const PreparedProgramme &programme) { return value < programme.start; });
        if (candidate == items->begin()) {
            return static_cast<const PreparedProgramme *>(nullptr);
        }
        const PreparedProgramme &previous = *std::prev(candidate);
        return time >= previous.start && time < previous.stop ? &previous : nullptr;
    };

    if (const PreparedProgramme *exact = exactProgramme(programmes, selectedTime)) {
        return exact;
    }
    if (programmes && selectedTime.isValid()) {
        const auto next = std::lower_bound(
            programmes->begin(), programmes->end(), selectedTime,
            [](const PreparedProgramme &programme, const QDateTime &value) { return programme.start < value; });
        if (next != programmes->end()) {
            return &*next;
        }
    }

    if (const PreparedProgramme *current = exactProgramme(programmes, QDateTime::currentDateTime())) {
        return current;
    }
    if (programmes && !programmes->empty()) {
        return &programmes->front();
    }
    if (!m_guide.programmes.empty()) {
        const XMLTVProgramme *first = &m_guide.programmes.front();
        const auto firstChannel = m_programmesByChannel.constFind(QString::fromStdString(first->channelId));
        if (firstChannel != m_programmesByChannel.cend()) {
            const auto match = std::find_if(
                firstChannel->begin(), firstChannel->end(),
                [first](const PreparedProgramme &prepared) { return prepared.programme == first; });
            if (match != firstChannel->end()) {
                return &*match;
            }
        }
    }
    return nullptr;
}

const GuideOverlay::PreparedProgramme *GuideOverlay::nextProgrammeForChannel(
    const PreparedProgramme *current) const
{
    if (!current || !current->programme) {
        return nullptr;
    }
    const std::vector<PreparedProgramme> *programmes = programmesForChannel(current->programme->channelId);
    if (!programmes) {
        return nullptr;
    }
    const auto next = std::upper_bound(
        programmes->begin(), programmes->end(), current->start,
        [](const QDateTime &value, const PreparedProgramme &programme) { return value < programme.start; });
    return next == programmes->end() ? nullptr : &*next;
}

QDateTime GuideOverlay::clampedTimelineStart(QDateTime value) const
{
    const QDateTime minStart = guideStartTime();
    const QDateTime fallbackStop = minStart.addSecs(24 * 60 * 60);
    const QDateTime maxStop = !m_guideStop.isValid() || m_guideStop > fallbackStop ? fallbackStop : m_guideStop;
    const QDateTime maxStart = std::max(
        minStart, maxStop.addSecs(-(kVisibleHalfHourSlots * kHalfHourSeconds)));

    if (!value.isValid() || value < minStart) {
        return minStart;
    }
    return value > maxStart ? maxStart : value;
}

void GuideOverlay::setGuide(XMLTVGuide guide)
{
    m_guide = std::move(guide);
    prepareGuide();
    if (m_selectedChannelId == 0) {
        m_selectedChannelId = m_currentChannelId;
    }
    m_timelineStart = clampedTimelineStart(m_timelineStart);
    rebuildUi();
}

void GuideOverlay::setTextSize(int textSize)
{
    textSize = std::clamp(textSize, MinimumTextSize, MaximumTextSize);
    if (textSize == m_textSize) {
        return;
    }
    m_textSize = textSize;
    setStyleSheet(guideStyleSheet(m_textSize));
    // Before the first setGuide() there is nothing to lay out yet.
    if (!m_guide.channels.empty()) {
        rebuildUi();
    }
}

int GuideOverlay::textSize() const
{
    return m_textSize;
}

void GuideOverlay::setCurrentChannelId(int channelId)
{
    m_currentChannelId = channelId;
    m_selectedChannelId = channelId;
}

void GuideOverlay::resetTimelineToNow()
{
    m_timelineStart = guideStartTime();
    rebuildUi();
}

void GuideOverlay::scrollChannels(int offset)
{
    if (m_guide.channels.empty()) {
        return;
    }

    int index = selectedChannelIndex();
    if (index < 0) {
        index = 0;
    }
    index = std::clamp(index + offset, 0, static_cast<int>(m_guide.channels.size()) - 1);

    const QString id = QString::fromStdString(m_guide.channels.at(index).id);
    const QString prefix = QStringLiteral("retrotv-");
    if (id.startsWith(prefix)) {
        m_selectedChannelId = id.mid(prefix.size()).toInt();
    }

    keepSelectionVisible(visibleRowCapacity());
    scrollToSelectedChannel();
    updateSelectionStyles();
    updateSelectedProgrammeDetails();
}

void GuideOverlay::pageChannels(int direction)
{
    // Keep one row of overlap so the viewer does not lose their place.
    scrollChannels(direction * std::max(1, visibleRowCapacity() - 1));
}

void GuideOverlay::scrollTimeline(int offset)
{
    if (m_guide.programmes.empty() || offset == 0) {
        return;
    }

    const QDateTime nextStart = clampedTimelineStart(m_timelineStart.addSecs(offset * kHalfHourSeconds));
    if (nextStart == m_timelineStart) {
        return;
    }

    m_timelineStart = nextStart;
    updateTimelineWindow();
}

int GuideOverlay::selectedChannelIndex() const
{
    int index = channelIndexById(m_guide, m_selectedChannelId);
    if (index >= 0) {
        return index;
    }
    index = channelIndexById(m_guide, m_currentChannelId);
    if (index >= 0) {
        return index;
    }
    return m_guide.channels.empty() ? -1 : 0;
}

int GuideOverlay::selectedChannelId() const
{
    const int index = selectedChannelIndex();
    if (index < 0 || index >= static_cast<int>(m_guide.channels.size())) {
        return 0;
    }

    const QString id = QString::fromStdString(m_guide.channels.at(index).id);
    const QString prefix = QStringLiteral("retrotv-");
    if (!id.startsWith(prefix)) {
        return 0;
    }
    return id.mid(prefix.size()).toInt();
}

int GuideOverlay::visibleRowCapacity() const
{
    if (height() <= 0) {
        return kFallbackVisibleRows;
    }

    const GuideMetrics metrics = guideMetrics(m_textSize);
    const int reservedHeight = 380 + (metrics.headerHeight - 80) + (metrics.timeHeaderHeight - 40)
        + (metrics.detailHeight - 280) + (metrics.footerHeight - 60);
    const int availableHeight = std::max(kMinimumVisibleRows * metrics.rowHeight, height() - reservedHeight);
    return std::max(kMinimumVisibleRows, availableHeight / metrics.rowHeight);
}

void GuideOverlay::keepSelectionVisible(int visibleRows)
{
    if (m_guide.channels.empty()) {
        m_channelOffset = 0;
        return;
    }

    visibleRows = std::max(1, visibleRows);
    const int selectedIndex = std::max(0, selectedChannelIndex());
    if (selectedIndex < m_channelOffset) {
        m_channelOffset = selectedIndex;
    } else if (selectedIndex >= m_channelOffset + visibleRows) {
        m_channelOffset = selectedIndex - visibleRows + 1;
    }

    const int maxOffset = std::max(0, static_cast<int>(m_guide.channels.size()) - visibleRows);
    m_channelOffset = std::clamp(m_channelOffset, 0, maxOffset);
}

void GuideOverlay::scrollToSelectedChannel()
{
    if (!m_gridScrollArea) {
        return;
    }
    m_gridScrollArea->verticalScrollBar()->setValue(m_channelOffset * guideMetrics(m_textSize).rowHeight);
}

void GuideOverlay::updateSelectionStyles()
{
    if (!m_gridFrame) {
        return;
    }

    const int selectedIndex = selectedChannelIndex();
    const auto repolish = [](QWidget *widget) {
        widget->style()->unpolish(widget);
        widget->style()->polish(widget);
        widget->update();
    };

    for (QFrame *cell : m_gridFrame->findChildren<QFrame *>()) {
        if (!cell->property("channelIndex").isValid()) {
            continue;
        }
        const bool selected = cell->property("channelIndex").toInt() == selectedIndex;
        cell->setObjectName(selected ? QStringLiteral("selectedChannelCell") : QStringLiteral("channelCell"));
        repolish(cell);
    }

    for (QLabel *cell : m_gridFrame->findChildren<QLabel *>()) {
        if (!cell->property("channelIndex").isValid()) {
            continue;
        }
        const bool selected = cell->property("timelineSelection").toBool()
            && cell->property("channelIndex").toInt() == selectedIndex;
        const QString style = programCellStyle(selected, cell->property("currentlyAiring").toBool());
        if (cell->objectName() != style || !cell->property("styleApplied").toBool()) {
            cell->setObjectName(style);
            repolish(cell);
            cell->setProperty("styleApplied", true);
        }
    }
}

void GuideOverlay::updateSelectedProgrammeDetails()
{
    const PreparedProgramme *selectedPrepared = programmeAtTime(
        m_selectedChannelId > 0 ? m_selectedChannelId : m_currentChannelId,
        m_timelineStart);
    const PreparedProgramme *upNextPrepared = nextProgrammeForChannel(selectedPrepared);
    const XMLTVProgramme *selectedProgramme = selectedPrepared ? selectedPrepared->programme : nullptr;
    const XMLTVProgramme *upNextProgramme = upNextPrepared ? upNextPrepared->programme : nullptr;

    updatePosterLabel(m_detailPosterLabel, selectedProgramme);
    if (m_detailTitleLabel) {
        m_detailTitleLabel->setText(selectedProgramme
                                        ? QString::fromStdString(selectedProgramme->title)
                                        : QStringLiteral("No guide data"));
    }
    if (m_detailTimeLabel) {
        m_detailTimeLabel->setText(selectedProgramme
                                       ? QStringLiteral("%1 - %2")
                                             .arg(formatTime(selectedPrepared->start), formatTime(selectedPrepared->stop))
                                       : QString());
    }
    if (m_detailDescriptionLabel) {
        m_detailDescriptionLabel->setText(selectedProgramme
                                              ? QString::fromStdString(selectedProgramme->description)
                                              : QStringLiteral("data/guide.xml did not contain programme entries."));
    }
    if (m_upNextTitleLabel) {
        m_upNextTitleLabel->setText(upNextProgramme
                                        ? QString::fromStdString(upNextProgramme->title)
                                        : QStringLiteral("No later programme"));
    }
    if (m_upNextTimeLabel) {
        m_upNextTimeLabel->setText(upNextProgramme
                                       ? QStringLiteral("%1 - %2")
                                             .arg(formatTime(upNextPrepared->start), formatTime(upNextPrepared->stop))
                                       : QString());
    }
}

QLabel *GuideOverlay::acquireProgramCell(const XMLTVProgramme *programme, const QString &objectName)
{
    QLabel *cell = nullptr;
    if (m_reusableProgramCells.empty()) {
        cell = new QLabel(m_gridFrame);
        cell->setWordWrap(true);
        cell->setAlignment(Qt::AlignLeft | Qt::AlignVCenter);
    } else {
        cell = m_reusableProgramCells.back();
        m_reusableProgramCells.pop_back();
    }
    cell->setFixedHeight(guideMetrics(m_textSize).rowHeight);

    cell->setText(programme ? guideCellTitle(*programme) : QString());
    cell->setObjectName(objectName);
    cell->setProperty("styleApplied", false);
    cell->show();
    return cell;
}

void GuideOverlay::recycleTimelineCells()
{
    for (int index = m_gridLayout->count() - 1; index >= 0; --index) {
        QLayoutItem *item = m_gridLayout->itemAt(index);
        if (!item) {
            continue;
        }

        int row = 0;
        int column = 0;
        int rowSpan = 0;
        int columnSpan = 0;
        m_gridLayout->getItemPosition(index, &row, &column, &rowSpan, &columnSpan);
        if (column == 0) {
            continue;
        }

        item = m_gridLayout->takeAt(index);
        if (auto *cell = qobject_cast<QLabel *>(item->widget())) {
            cell->hide();
            cell->setProperty("channelIndex", QVariant());
            cell->setProperty("timelineSelection", QVariant());
            cell->setProperty("currentlyAiring", QVariant());
            cell->setProperty("styleApplied", false);
            m_reusableProgramCells.push_back(cell);
        } else if (QWidget *widget = item->widget()) {
            delete widget;
        }
        delete item;
    }
}

void GuideOverlay::updateTimelineWindow()
{
    if (!m_gridLayout) {
        rebuildUi();
        return;
    }

    const QDateTime timelineStart = m_timelineStart;
    const QDateTime timelineStop = timelineStart.addSecs(kVisibleHalfHourSlots * kHalfHourSeconds);
    const QDateTime now = QDateTime::currentDateTime();
    m_gridFrame->setUpdatesEnabled(false);

    for (int slot = 0; slot < static_cast<int>(m_timeHeaderLabels.size()); ++slot) {
        if (!m_timeHeaderLabels.at(slot)) {
            continue;
        }
        m_timeHeaderLabels.at(slot)->setText(
            timelineStart.addSecs(slot * kHalfHourSeconds).time().toString(QStringLiteral("h:mm AP")));
    }

    recycleTimelineCells();

    const int selectedIndex = selectedChannelIndex();
    for (int row = 0; row < static_cast<int>(m_guide.channels.size()); ++row) {
        const XMLTVChannel &channel = m_guide.channels.at(row);
        const bool selectedRow = row == selectedIndex;
        int channelId = 0;
        const QString channelIdText = QString::fromStdString(channel.id);
        const QString channelPrefix = QStringLiteral("retrotv-");
        if (channelIdText.startsWith(channelPrefix)) {
            channelId = channelIdText.mid(channelPrefix.size()).toInt();
        }

        const PreparedProgramme *rowTimelineProgramme = programmeAtTime(channelId, m_timelineStart);
        bool hasVisibleProgramme = false;
        const std::vector<PreparedProgramme> *programmes = programmesForChannel(channel.id);
        if (programmes) {
            for (const PreparedProgramme &prepared : *programmes) {
                if (prepared.start >= timelineStop) {
                    break;
                }
                if (prepared.stop <= timelineStart) {
                    continue;
                }

                const QDateTime clippedStart = prepared.start < timelineStart ? timelineStart : prepared.start;
                const QDateTime clippedStop = prepared.stop > timelineStop ? timelineStop : prepared.stop;
                const int column = 1 + std::max(0, minutesBetween(timelineStart, clippedStart) / 30);
                const int span = std::min(kVisibleHalfHourSlots - (column - 1), halfHourSpan(clippedStart, clippedStop));
                const bool timelineSelection = rowTimelineProgramme == &prepared;
                const bool currentlyAiring = now >= prepared.start && now < prepared.stop;
                const QString style = programCellStyle(selectedRow && timelineSelection, currentlyAiring);
                QLabel *cell = acquireProgramCell(prepared.programme, style);
                cell->setProperty("channelIndex", row);
                cell->setProperty("timelineSelection", timelineSelection);
                cell->setProperty("currentlyAiring", currentlyAiring);
                m_gridLayout->addWidget(cell, row, column, 1, span);
                hasVisibleProgramme = true;
            }
        }

        if (!hasVisibleProgramme) {
            QLabel *cell = acquireProgramCell(nullptr, QStringLiteral("programCell"));
            cell->setProperty("channelIndex", row);
            cell->setProperty("timelineSelection", false);
            cell->setProperty("currentlyAiring", false);
            m_gridLayout->addWidget(cell, row, 1, 1, kVisibleHalfHourSlots);
        }
    }

    for (int row = static_cast<int>(m_guide.channels.size()); row < m_gridLayout->rowCount(); ++row) {
        QLabel *cell = acquireProgramCell(nullptr, QStringLiteral("programCell"));
        cell->setProperty("channelIndex", row);
        cell->setProperty("timelineSelection", false);
        cell->setProperty("currentlyAiring", false);
        m_gridLayout->addWidget(cell, row, 1, 1, kVisibleHalfHourSlots);
    }

    updateSelectionStyles();
    updateSelectedProgrammeDetails();
    m_gridFrame->setUpdatesEnabled(true);
    m_gridFrame->update();
}

void GuideOverlay::rebuildUi()
{
    const GuideMetrics metrics = guideMetrics(m_textSize);
    m_reusableProgramCells.clear();
    m_gridLayout = nullptr;
    m_timeHeaderLabels.clear();
    m_detailPosterLabel = nullptr;
    m_detailTitleLabel = nullptr;
    m_detailTimeLabel = nullptr;
    m_detailDescriptionLabel = nullptr;
    m_upNextTitleLabel = nullptr;
    m_upNextTimeLabel = nullptr;

    auto *root = qobject_cast<QVBoxLayout *>(layout());
    if (root) {
        clearLayout(root);
    } else {
        root = new QVBoxLayout(this);
    }
    root->setContentsMargins(28, 8, 28, 8);
    root->setSpacing(4);

    m_timelineStart = clampedTimelineStart(m_timelineStart);
    const QDateTime now = QDateTime::currentDateTime();
    const PreparedProgramme *selectedPrepared = programmeAtTime(
        m_selectedChannelId > 0 ? m_selectedChannelId : m_currentChannelId,
        m_timelineStart);
    const XMLTVProgramme *selectedProgramme = selectedPrepared ? selectedPrepared->programme : nullptr;
    const QString selectedStart = now.time().toString(QStringLiteral("h:mm AP"));
    const QString selectedDate = now.date().toString(QStringLiteral("ddd, MMM d, yyyy"));

    auto *headerFrame = new QFrame(this);
    headerFrame->setFixedHeight(metrics.headerHeight);
    auto *header = new QHBoxLayout(headerFrame);
    header->setContentsMargins(0, 0, 0, 0);
    header->setSpacing(24);
    auto *brandRow = new QHBoxLayout();
    brandRow->setSpacing(0);
    brandRow->addWidget(label(QStringLiteral("Retro"), QStringLiteral("brand")));
    brandRow->addWidget(label(QStringLiteral("TV"), QStringLiteral("brandRed")));
    header->addLayout(brandRow);
    auto *divider = new QFrame(this);
    divider->setFrameShape(QFrame::VLine);
    divider->setFixedHeight(metrics.dividerHeight);
    divider->setStyleSheet(QStringLiteral("color: rgba(255, 255, 255, 50);"));
    header->addWidget(divider);
    header->addWidget(label(QStringLiteral("TV Guide"), QStringLiteral("screenTitle")));
    header->addStretch(1);
    auto *timeColumn = new QVBoxLayout();
    timeColumn->setSpacing(2);
    timeColumn->addWidget(label(selectedStart, QStringLiteral("clock")), 0, Qt::AlignRight);
    timeColumn->addWidget(label(selectedDate, QStringLiteral("date")), 0, Qt::AlignRight);
    header->addLayout(timeColumn);
    root->addWidget(headerFrame, 0);

    QDateTime timelineStart = m_timelineStart;
    const QDateTime timelineStop = timelineStart.addSecs(kVisibleHalfHourSlots * 30 * 60);

    auto *timeHeaderFrame = new QFrame(this);
    timeHeaderFrame->setFixedHeight(metrics.timeHeaderHeight);
    auto *timeHeader = new QHBoxLayout(timeHeaderFrame);
    timeHeader->setContentsMargins(0, 0, 0, 0);
    timeHeader->setSpacing(0);
    auto *channelsHeader = label(QStringLiteral("CHANNELS"), QStringLiteral("axisLabel"));
    channelsHeader->setFixedWidth(metrics.channelColumnWidth);
    timeHeader->addWidget(channelsHeader);
    for (int slot = 0; slot < kVisibleHalfHourSlots; ++slot) {
        auto *timeLabel = label(timelineStart.addSecs(slot * 30 * 60).time().toString(QStringLiteral("h:mm AP")),
                                QStringLiteral("axisLabel"));
        timeLabel->setAlignment(Qt::AlignCenter);
        timeHeader->addWidget(timeLabel, 1);
        m_timeHeaderLabels.push_back(timeLabel);
    }
    root->addWidget(timeHeaderFrame, 0);

    m_gridFrame = new QFrame(this);
    m_gridFrame->setObjectName(QStringLiteral("gridFrame"));
    auto *grid = new QGridLayout(m_gridFrame);
    m_gridLayout = grid;
    grid->setContentsMargins(0, 0, 0, 0);
    grid->setSpacing(0);

    const int visibleRows = std::min(
        visibleRowCapacity(),
        std::max(kMinimumVisibleRows, static_cast<int>(m_guide.channels.size())));
    keepSelectionVisible(visibleRows);
    const int totalRows = std::max(visibleRows, static_cast<int>(m_guide.channels.size()));

    for (int row = 0; row < totalRows; ++row) {
        grid->setRowMinimumHeight(row, metrics.rowHeight);
        grid->setRowStretch(row, 0);
    }

    const int selectedIndex = selectedChannelIndex();

    for (int row = 0; row < static_cast<int>(m_guide.channels.size()); ++row) {
        const XMLTVChannel &channel = m_guide.channels.at(row);
        const bool selectedRow = row == selectedIndex;
        int channelId = 0;
        const QString channelIdText = QString::fromStdString(channel.id);
        const QString channelPrefix = QStringLiteral("retrotv-");
        if (channelIdText.startsWith(channelPrefix)) {
            channelId = channelIdText.mid(channelPrefix.size()).toInt();
        }
        const PreparedProgramme *rowTimelineProgramme = programmeAtTime(channelId, m_timelineStart);
        QWidget *channelWidget = channelCellWidget(
                            channel,
                            row + 1,
                            selectedRow ? QStringLiteral("selectedChannelCell") : QStringLiteral("channelCell"),
                            metrics);
        channelWidget->setProperty("channelIndex", row);
        grid->addWidget(channelWidget, row, 0);
        bool hasVisibleProgramme = false;

        const std::vector<PreparedProgramme> *programmes = programmesForChannel(channel.id);
        if (programmes) {
            for (const PreparedProgramme &prepared : *programmes) {
                if (prepared.start >= timelineStop) {
                    break;
                }
                if (prepared.stop <= timelineStart) {
                    continue;
                }

                const QDateTime clippedStart = prepared.start < timelineStart ? timelineStart : prepared.start;
                const QDateTime clippedStop = prepared.stop > timelineStop ? timelineStop : prepared.stop;
                const int column = 1 + std::max(0, minutesBetween(timelineStart, clippedStart) / 30);
                const int span = std::min(kVisibleHalfHourSlots - (column - 1), halfHourSpan(clippedStart, clippedStop));
                const bool timelineSelection = rowTimelineProgramme == &prepared;
                const bool currentlyAiring = now >= prepared.start && now < prepared.stop;
                const QString style = programCellStyle(selectedRow && timelineSelection, currentlyAiring);
                QLabel *cell = acquireProgramCell(prepared.programme, style);
                cell->setProperty("channelIndex", row);
                cell->setProperty("timelineSelection", timelineSelection);
                cell->setProperty("currentlyAiring", currentlyAiring);
                grid->addWidget(cell, row, column, 1, span);
                hasVisibleProgramme = true;
            }
        }

        if (!hasVisibleProgramme) {
            QLabel *cell = acquireProgramCell(nullptr, QStringLiteral("programCell"));
            cell->setProperty("channelIndex", row);
            cell->setProperty("timelineSelection", false);
            cell->setProperty("currentlyAiring", false);
            grid->addWidget(cell, row, 1, 1, kVisibleHalfHourSlots);
        }
    }

    for (int row = static_cast<int>(m_guide.channels.size()); row < totalRows; ++row) {
        auto *emptyChannel = new QFrame();
        emptyChannel->setObjectName(QStringLiteral("channelCell"));
        emptyChannel->setFixedHeight(metrics.rowHeight);
        auto *emptyLayout = new QHBoxLayout(emptyChannel);
        emptyLayout->setContentsMargins(18, 0, 12, 0);
        auto *number = label(QString::number(row + 1), QStringLiteral("channelNumber"));
        number->setAlignment(Qt::AlignLeft | Qt::AlignVCenter);
        emptyLayout->addWidget(number);
        grid->addWidget(emptyChannel, row, 0);

        QLabel *cell = acquireProgramCell(nullptr, QStringLiteral("programCell"));
        cell->setProperty("channelIndex", row);
        cell->setProperty("timelineSelection", false);
        cell->setProperty("currentlyAiring", false);
        grid->addWidget(cell, row, 1, 1, kVisibleHalfHourSlots);
    }

    int screenWidth = 0;
    if (QScreen *screen = QGuiApplication::primaryScreen()) {
        screenWidth = screen->geometry().width();
    }
    const int parentWidth = parentWidget() ? parentWidget()->width() : 0;
    const int availableGridWidth = std::max({1280, width() - 56, parentWidth - 56, screenWidth - 56});
    const int timelineColumnWidth = std::max(160, (availableGridWidth - metrics.channelColumnWidth) / kVisibleHalfHourSlots);
    const int gridWidth = metrics.channelColumnWidth + (timelineColumnWidth * kVisibleHalfHourSlots);

    grid->setColumnMinimumWidth(0, metrics.channelColumnWidth);
    grid->setColumnStretch(0, 0);
    for (int column = 1; column <= kVisibleHalfHourSlots; ++column) {
        grid->setColumnMinimumWidth(column, timelineColumnWidth);
        grid->setColumnStretch(column, 1);
    }

    m_gridFrame->setMinimumWidth(gridWidth);
    m_gridFrame->setFixedWidth(gridWidth);
    m_gridFrame->setFixedHeight((totalRows * metrics.rowHeight) + 2);
    m_gridScrollArea = new QScrollArea(this);
    m_gridScrollArea->setObjectName(QStringLiteral("guideScrollArea"));
    m_gridScrollArea->setWidget(m_gridFrame);
    m_gridScrollArea->setWidgetResizable(false);
    m_gridScrollArea->setHorizontalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    m_gridScrollArea->setVerticalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    m_gridScrollArea->setFrameShape(QFrame::NoFrame);
    m_gridScrollArea->setMinimumHeight((kMinimumVisibleRows * metrics.rowHeight) + 2);
    root->addWidget(m_gridScrollArea, 1);
    scrollToSelectedChannel();

    auto *detail = new QFrame(this);
    detail->setObjectName(QStringLiteral("detailPanel"));
    detail->setFixedHeight(metrics.detailHeight);
    auto *detailLayout = new QHBoxLayout(detail);
    detailLayout->setContentsMargins(12, 10, 24, 10);
    detailLayout->setSpacing(28);

    const QString detailTitle = selectedProgramme
                                    ? QString::fromStdString(selectedProgramme->title)
                                    : QStringLiteral("No guide data");
    const QString detailDescription = selectedProgramme
                                          ? QString::fromStdString(selectedProgramme->description)
                                          : QStringLiteral("data/guide.xml did not contain programme entries.");
    const QString detailTime = selectedProgramme
                                   ? QStringLiteral("%1 - %2").arg(formatTime(selectedPrepared->start), formatTime(selectedPrepared->stop))
                                   : QString();

    m_detailPosterLabel = posterLabel(selectedProgramme);
    detailLayout->addWidget(m_detailPosterLabel);

    auto *detailText = new QVBoxLayout();
    detailText->setSpacing(8);
    m_detailTitleLabel = label(detailTitle, QStringLiteral("detailTitle"));
    m_detailTimeLabel = label(detailTime, QStringLiteral("detailMeta"));
    m_detailDescriptionLabel = label(detailDescription, QStringLiteral("description"));
    detailText->addWidget(m_detailTitleLabel);
    detailText->addWidget(m_detailTimeLabel);
    detailText->addWidget(m_detailDescriptionLabel);
    detailText->addStretch(1);
    detailLayout->addLayout(detailText, 1);

    const PreparedProgramme *upNextPrepared = nextProgrammeForChannel(selectedPrepared);
    const XMLTVProgramme *upNextProgramme = upNextPrepared ? upNextPrepared->programme : nullptr;
    auto *upNext = new QVBoxLayout();
    upNext->setSpacing(8);
    upNext->addStretch(1);
    upNext->addWidget(label(QStringLiteral("UP NEXT"), QStringLiteral("axisLabel")));
    m_upNextTitleLabel = label(upNextProgramme
                                   ? QString::fromStdString(upNextProgramme->title)
                                   : QStringLiteral("No later programme"));
    m_upNextTimeLabel = label(upNextProgramme
                                  ? QStringLiteral("%1 - %2")
                                        .arg(formatTime(upNextPrepared->start), formatTime(upNextPrepared->stop))
                                  : QString(),
                              QStringLiteral("detailMeta"));
    upNext->addWidget(m_upNextTitleLabel);
    upNext->addWidget(m_upNextTimeLabel);
    detailLayout->addLayout(upNext);

    root->addWidget(detail, 0);

    auto *footerFrame = new QFrame(this);
    footerFrame->setFixedHeight(metrics.footerHeight);
    auto *footer = new QHBoxLayout(footerFrame);
    footer->setContentsMargins(50, 6, 20, 6);
    footer->setSpacing(64);
    addFooterAction(footer, QStringLiteral("+"), QStringLiteral("NAVIGATE"));
    addFooterAction(footer, QStringLiteral("OK"), QStringLiteral("WATCH LIVE"));
    addFooterAction(footer, QStringLiteral("="), QStringLiteral("OPTIONS"));
    addFooterAction(footer, QString(), QStringLiteral("JUMP TO NOW"), QStringLiteral("yellowIcon"));
    addFooterAction(footer, QString(), QStringLiteral("FILTER"), QStringLiteral("blueIcon"));
    addFooterAction(footer, QString(), QStringLiteral("EXIT GUIDE"), QStringLiteral("redIcon"));
    footer->addStretch(1);
    root->addWidget(footerFrame, 0);
    updateSelectionStyles();
}

void GuideOverlay::keyPressEvent(QKeyEvent *event)
{
    if (event->key() == Qt::Key_Escape) {
        emit dismissRequested();
        event->accept();
        return;
    }
    if (event->key() == Qt::Key_Down) {
        scrollChannels(1);
        event->accept();
        return;
    }
    if (event->key() == Qt::Key_Up) {
        scrollChannels(-1);
        event->accept();
        return;
    }
    if (event->key() == Qt::Key_Right) {
        scrollTimeline(1);
        event->accept();
        return;
    }
    if (event->key() == Qt::Key_Left) {
        scrollTimeline(-1);
        event->accept();
        return;
    }
    if (event->key() == Qt::Key_Enter || event->key() == Qt::Key_Return) {
        emit channelSelected(selectedChannelId());
        event->accept();
        return;
    }

    QWidget::keyPressEvent(event);
}
