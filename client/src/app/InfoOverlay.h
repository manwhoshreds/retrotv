#pragma once

#include <QWidget>
#include <QDateTime>
#include <QString>

class QKeyEvent;
class QLabel;
class QNetworkAccessManager;
class QNetworkReply;
class QProgressBar;
class QTimer;
class ThreeLineLabel;

struct NowPlayingInfo
{
    int channelNumber = 0;
    QString channelName;
    QString title;
    QString timeRange;
    QString description;
    QString posterPath;
    QString upNextTitle;
    QString upNextTimeRange;
    QDateTime startTime;
    QDateTime endTime;
    int year = 0;
    QString rating;
    QString genre;
    int seasonNumber = 0;
    int episodeNumber = 0;
    bool hasClosedCaptions = false;
    bool isHD = false;
};

class InfoOverlay : public QWidget
{
    Q_OBJECT

public:
    explicit InfoOverlay(QWidget *parent = nullptr);
    void setNowPlaying(const NowPlayingInfo &info);

signals:
    void dismissRequested();

protected:
    void keyPressEvent(QKeyEvent *event) override;

private:
    void loadPoster(const QString &path, const QString &fallback);
    void updateProgress();

    QLabel *m_channelNumberLabel = nullptr;
    QLabel *m_channelNameLabel = nullptr;
    QLabel *m_titleLabel = nullptr;
    QLabel *m_timeLabel = nullptr;
    ThreeLineLabel *m_descriptionLabel = nullptr;
    QLabel *m_posterLabel = nullptr;
    QLabel *m_ccBadge = nullptr;
    QLabel *m_hdBadge = nullptr;
    QLabel *m_clockLabel = nullptr;
    QLabel *m_upNextTitleLabel = nullptr;
    QLabel *m_upNextTimeLabel = nullptr;
    QLabel *m_remainingLabel = nullptr;
    QLabel *m_yearRatingLabel = nullptr;
    QLabel *m_genreEpisodeLabel = nullptr;
    QLabel *m_yearIconLabel = nullptr;
    QLabel *m_genreIconLabel = nullptr;
    QProgressBar *m_progressBar = nullptr;
    QTimer *m_progressTimer = nullptr;
    QDateTime m_programStart;
    QDateTime m_programEnd;
    QNetworkAccessManager *m_networkManager = nullptr;
    QNetworkReply *m_posterReply = nullptr;
};
