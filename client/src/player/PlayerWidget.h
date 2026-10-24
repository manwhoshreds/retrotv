#pragma once

#include <cstdint>

#include <QString>
#include <QOpenGLWidget>
#include <QTimer>

#include <mpv/client.h>
#include <mpv/render.h>

class PlayerWidget : public QOpenGLWidget
{
    Q_OBJECT

public:
    explicit PlayerWidget(QWidget *parent = nullptr);
    ~PlayerWidget() override;

    void play(const char *filename);
    void stop();
    void loadMedia(const QString &path, int startSeconds = 0);
    void handleMpvEvents();
    void adjustVolume(int delta);
    void toggleMute();

signals:
    void mediaStatus(QString message);
    void mediaLoaded(QString path);
    void mediaFailed(QString path, QString message);
    void mediaEnded(QString path);

private:
    void initializeGL() override;
    void paintGL() override;
    void applyAudioDefaults();
    void reportAudioState();
    void reportPlaybackStats();

    mpv_handle *m_mpv = nullptr;
    mpv_render_context *m_mpvRenderContext = nullptr;
    bool m_waitingForPlaybackRestart = false;
    int m_pendingStartSeconds = 0;
    QString m_currentPath;
    QString m_lastPlaybackErrorDetail;
    QTimer m_statsTimer;
    int m_volume = 100;
    bool m_muted = false;
    int64_t m_lastDroppedFrames = 0;
    int64_t m_lastDecoderDroppedFrames = 0;
    int m_openFilesHighWater = 0;
};
