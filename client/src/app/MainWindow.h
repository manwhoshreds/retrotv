#pragma once

#include <optional>
#include <vector>

#include <QFutureWatcher>
#include <QDateTime>
#include <QList>
#include <QMainWindow>
#include <QPixmap>
#include <QString>

#include "app/InfoOverlay.h"
#include "models/Channel.h"
#include "network/ServerDiscovery.h"

class GuideOverlay;
class ChannelTransitionOverlay;
class QEvent;
class QLabel;
class PlayerWidget;
class QPushButton;
class QKeyEvent;
class QMoveEvent;
class QNetworkAccessManager;
class QNetworkReply;
class QResizeEvent;
class QStackedLayout;
class SettingsPage;
class QTimer;
class QWidget;

struct PlaybackTarget
{
    QString streamURL;
    int seekSeconds = 0;
    QString title;
    QString description;
    QString posterPath;
    QString timeRange;
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

class MainWindow : public QMainWindow
{
    Q_OBJECT

public:
    explicit MainWindow(QWidget *parent = nullptr);

protected:
    bool eventFilter(QObject *watched, QEvent *event) override;
    void keyPressEvent(QKeyEvent *event) override;
    void moveEvent(QMoveEvent *event) override;
    void resizeEvent(QResizeEvent *event) override;
    void changeEvent(QEvent *event) override;

private:
    enum class StartupState {
        Starting,
        Offline,
        NoChannels,
        LoadingGuide,
        Tuning,
        Playing,
        MediaUnavailable,
    };

    void startClient();
    void refreshGuide();
    void showSettings();
    void hideSettings();
    void setStartupState(StartupState state, const QString &title = {}, const QString &detail = {});
    void tuneChannel(int channelIndex, bool showChannelTransition = true);
    void tuneChannelId(int channelId);
    void tuneRelative(int offset);
    bool handleChannelNumberKey(QKeyEvent *event);
    bool handleRemoteKey(QKeyEvent *event);
    void appendChannelNumberDigit(int digit);
    void commitChannelNumber();
    void cancelChannelNumber();
    int maximumChannelNumberDigits() const;
    void applyPlaybackTarget(const Channel &channel, const PlaybackTarget &target, bool useTransition);
    void showNoCurrentProgram(const Channel &channel);
    void showMediaUnavailable(const Channel &channel, const QString &message);
    void showChannelTransition(const Channel &channel, const QString &title = {}, const QString &timeRange = {});
    void showInfoOverlay();
    void showGuideOverlay();
    void hideOverlays();
    void toggleFullscreen();
    void updateOverlayGeometry();
    void loadChannelBugLogo(const QString &logoURL, int tuneGeneration);
    void updateChannelBugGeometry();
    void updateChannelBugVisibility();

    QWidget *m_rootWidget = nullptr;
    PlayerWidget *m_playerWidget = nullptr;
    QNetworkAccessManager *m_tuneNetworkManager = nullptr;
    QNetworkReply *m_tuneReply = nullptr;
    QTimer *m_channelNumberTimer = nullptr;
    QTimer *m_guideRefreshTimer = nullptr;
    bool m_guideRefreshInFlight = false;
    QWidget *m_stateView = nullptr;
    QStackedLayout *m_stateStack = nullptr;
    QWidget *m_genericStateView = nullptr;
    QWidget *m_offlineView = nullptr;
    QLabel *m_stateLabel = nullptr;
    QPushButton *m_retryButton = nullptr;
    QPushButton *m_settingsButton = nullptr;
    SettingsPage *m_settingsPage = nullptr;
    InfoOverlay *m_infoOverlay = nullptr;
    GuideOverlay *m_guideOverlay = nullptr;
    ChannelTransitionOverlay *m_channelTransitionOverlay = nullptr;
    QLabel *m_channelBugLabel = nullptr;
    QPixmap m_channelBugPixmap;
    std::vector<Channel> m_channels;
    QString m_channelNumberBuffer;
    QString m_backendURL;
    QString m_requestedServerURL;
    QString m_discoveredBackendURL;
    QList<DiscoveredServer> m_discoveredServers;
    bool m_backendConnected = false;
    int m_currentChannelIndex = 0;
    int m_tuneGeneration = 0;
    bool m_restoreStateViewAfterGuide = false;
    bool m_waitingForSchedule = false;
    NowPlayingInfo m_nowPlayingInfo;
    StartupState m_startupState = StartupState::Starting;
    bool m_cursorHidden = false;
};
