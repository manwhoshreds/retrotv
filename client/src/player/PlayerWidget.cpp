#include "PlayerWidget.h"

#include <algorithm>
#include <cstdint>
#include <functional>

#include <climits>
#include <unistd.h>

#include <QByteArray>
#include <QDebug>
#include <QDir>
#include <QFile>
#include <QMap>
#include <QRegularExpression>
#include <QMetaObject>
#include <QOpenGLContext>
#include <QString>

#include <mpv/render_gl.h>

namespace {

void wakeup(void *context)
{
    auto *widget = static_cast<PlayerWidget *>(context);
    QMetaObject::invokeMethod(widget, [widget]() {
        widget->handleMpvEvents();
    }, Qt::QueuedConnection);
}

void renderUpdate(void *context)
{
    auto *widget = static_cast<PlayerWidget *>(context);
    QMetaObject::invokeMethod(widget, [widget]() {
        widget->update();
    }, Qt::QueuedConnection);
}

// Summarises this process's open file descriptors by target type, most
// common first, e.g. "socket:[N]=40 /dev/dri/renderDN=12".
QString openFileSummary()
{
    static const QRegularExpression digits(QStringLiteral("[0-9]+"));
    QMap<QString, int> counts;
    const QDir fdDir(QStringLiteral("/proc/self/fd"));
    for (const QString &entry : fdDir.entryList(QDir::AllEntries | QDir::System | QDir::NoDotAndDotDot)) {
        // QFileInfo resolves non-path targets such as "socket:[12]" against
        // the link's directory, so read the raw link text instead.
        char buffer[PATH_MAX];
        const ssize_t length = readlink(QFile::encodeName(fdDir.filePath(entry)).constData(), buffer, sizeof(buffer));
        if (length <= 0) {
            continue;
        }
        QString target = QString::fromLocal8Bit(buffer, length);
        target.replace(digits, QStringLiteral("N"));
        ++counts[target];
    }

    QList<QPair<int, QString>> sorted;
    for (auto it = counts.cbegin(); it != counts.cend(); ++it) {
        sorted.append({it.value(), it.key()});
    }
    std::sort(sorted.begin(), sorted.end(), std::greater<>());

    QStringList parts;
    for (qsizetype i = 0; i < sorted.size() && i < 6; ++i) {
        parts.append(QStringLiteral("%1=%2").arg(sorted.at(i).second).arg(sorted.at(i).first));
    }
    return parts.join(QLatin1Char(' '));
}

// Stand-in for glFenceSync handed to mpv; see getProcAddress. GLsync is an
// opaque pointer and GLenum/GLbitfield are unsigned ints, so this matches the
// real signature without needing the GLES3 headers.
void *noFenceSync(unsigned int, unsigned int)
{
    return nullptr;
}

void *getProcAddress(void *, const char *name)
{
    // mpv before 0.42 creates a GL fence after every libmpv frame but only
    // deletes them in a swap path libmpv never runs (mpv issue #17217). On the
    // Pi's v3d driver each fence holds a sync_file descriptor, so playback
    // exhausts the fd limit within minutes. mpv requires glFenceSync to exist,
    // so hand it a stub that creates no fence; mpv skips null fences, and
    // libmpv does not need them since Qt presents.
    const QByteArray function(name);
    if (function == "glFenceSync") {
        return reinterpret_cast<void *>(&noFenceSync);
    }

    QOpenGLContext *context = QOpenGLContext::currentContext();
    if (!context) {
        return nullptr;
    }
    const QFunctionPointer pointer = context->getProcAddress(function);
    return reinterpret_cast<void *>(pointer);
}

}

PlayerWidget::PlayerWidget(
    QWidget* parent
)
    : QOpenGLWidget(parent)
{
    setAttribute(Qt::WA_OpaquePaintEvent);
    setAutoFillBackground(false);

    m_statsTimer.setInterval(5000);
    connect(&m_statsTimer, &QTimer::timeout, this, &PlayerWidget::reportPlaybackStats);
}

void PlayerWidget::initializeGL()
{
    m_mpv = mpv_create();

    if (!m_mpv)
    {
        return;
    }

    // The render API does not select its video output automatically. Without
    // this explicit pre-initialization option mpv falls back to gpu-next and
    // creates an independent OS window.
    mpv_set_option_string(m_mpv, "vo", "libmpv");
    // Zero-copy hwdec (drm) fails on the Pi with "Failed to export gem bo to
    // dmabuf" and drops nearly every frame. The copy variants still decode in
    // hardware but hand frames to the renderer through system memory.
    // drm-copy covers the Pi's HEVC decoder; the Pi 4's H.264 decoder is only
    // reachable through v4l2m2m, which "auto-copy" never tries. Anything
    // neither handles (e.g. MPEG-4, 10-bit H.264) falls back to software.
    if (const int result = mpv_set_option_string(m_mpv, "hwdec", "drm-copy,v4l2m2m-copy,auto-copy"); result < 0) {
        qWarning() << "MPV rejected the hwdec list, using auto-copy:" << mpv_error_string(result);
        mpv_set_option_string(m_mpv, "hwdec", "auto-copy");
    }
    mpv_set_option_string(m_mpv, "vd-lavc-threads", "0");
    mpv_set_option_string(m_mpv, "video-sync", "audio");
    mpv_set_option_string(m_mpv, "interpolation", "no");
    // When a frame is shown more than once, mpv redraws it from a cached
    // texture whose area outside the video was never written; on the Pi that
    // copies stale GPU memory over the letterbox bars. Dumb mode skips that
    // cache (and the costlier scalers, which the Pi can do without).
    mpv_set_option_string(m_mpv, "gpu-dumb-mode", "yes");
    mpv_set_option_string(m_mpv, "hr-seek", "no");
    mpv_set_option_string(m_mpv, "cache", "yes");
    mpv_set_option_string(m_mpv, "cache-pause", "no");
    mpv_set_option_string(m_mpv, "demuxer-max-bytes", "64MiB");
    mpv_set_option_string(m_mpv, "demuxer-readahead-secs", "10");
    mpv_set_option_string(m_mpv, "audio", "auto");
    mpv_set_option_string(m_mpv, "aid", "auto");
    mpv_set_option_string(m_mpv, "mute", "no");
    mpv_set_option_string(m_mpv, "volume", "100");
    mpv_set_option_string(m_mpv, "audio-client-name", "RetroTV");
    mpv_set_option_string(m_mpv, "terminal", "no");
    mpv_set_option_string(m_mpv, "input-terminal", "no");
    mpv_set_option_string(m_mpv, "input-default-bindings", "no");
    mpv_set_option_string(m_mpv, "input-vo-keyboard", "no");
    mpv_set_option_string(m_mpv, "input-cursor", "no");

    // Extra mpv options for on-device experiments, applied after the defaults
    // above so they win: RETROTV_MPV_OPTIONS="name=value;name=value".
    const QStringList extraOptions = qEnvironmentVariable("RETROTV_MPV_OPTIONS").split(QLatin1Char(';'), Qt::SkipEmptyParts);
    for (const QString &option : extraOptions) {
        const qsizetype separator = option.indexOf(QLatin1Char('='));
        const QByteArray name = option.left(separator).trimmed().toUtf8();
        const QByteArray value = separator < 0 ? QByteArrayLiteral("yes") : option.mid(separator + 1).trimmed().toUtf8();
        const int result = mpv_set_option_string(m_mpv, name.constData(), value.constData());
        qWarning() << "RetroTV mpv option" << name << "=" << value << mpv_error_string(result);
    }

    mpv_set_wakeup_callback(m_mpv, wakeup, this);
    // RETROTV_MPV_LOG_LEVEL=v also logs mpv's GL feature detection.
    const QByteArray mpvLogLevel = qEnvironmentVariable("RETROTV_MPV_LOG_LEVEL", QStringLiteral("info")).toUtf8();
    mpv_request_log_messages(m_mpv, mpvLogLevel.constData());
    const int initResult = mpv_initialize(m_mpv);
    if (initResult < 0) {
        const QString message = QStringLiteral("MPV initialization failed: %1").arg(QString::fromUtf8(mpv_error_string(initResult)));
        qWarning() << message;
        emit mediaFailed(QString(), message);
        return;
    }

    mpv_opengl_init_params glInit{
        .get_proc_address = getProcAddress,
        .get_proc_address_ctx = nullptr,
    };
    mpv_render_param params[] = {
        {MPV_RENDER_PARAM_API_TYPE, const_cast<char *>(MPV_RENDER_API_TYPE_OPENGL)},
        {MPV_RENDER_PARAM_OPENGL_INIT_PARAMS, &glInit},
        {MPV_RENDER_PARAM_INVALID, nullptr},
    };
    const int renderResult = mpv_render_context_create(&m_mpvRenderContext, m_mpv, params);
    if (renderResult < 0) {
        const QString message = QStringLiteral("MPV render initialization failed: %1")
            .arg(QString::fromUtf8(mpv_error_string(renderResult)));
        qWarning() << message;
        emit mediaFailed(QString(), message);
        return;
    }
    mpv_render_context_set_update_callback(m_mpvRenderContext, renderUpdate, this);
}

void PlayerWidget::paintGL()
{
    if (!m_mpvRenderContext) {
        return;
    }
    const qreal scale = devicePixelRatioF();
    mpv_opengl_fbo framebuffer{
        .fbo = static_cast<int>(defaultFramebufferObject()),
        .w = qRound(width() * scale),
        .h = qRound(height() * scale),
        .internal_format = 0,
    };

    int flipY = 1;
    mpv_render_param params[] = {
        {MPV_RENDER_PARAM_OPENGL_FBO, &framebuffer},
        {MPV_RENDER_PARAM_FLIP_Y, &flipY},
        {MPV_RENDER_PARAM_INVALID, nullptr},
    };
    mpv_render_context_render(m_mpvRenderContext, params);
    mpv_render_context_report_swap(m_mpvRenderContext);
}

PlayerWidget::~PlayerWidget()
{
    if (m_mpvRenderContext) {
        makeCurrent();
        mpv_render_context_set_update_callback(m_mpvRenderContext, nullptr, nullptr);
        mpv_render_context_free(m_mpvRenderContext);
        m_mpvRenderContext = nullptr;
        doneCurrent();
    }
    if (m_mpv)
    {
        mpv_set_wakeup_callback(m_mpv, nullptr, nullptr);
        mpv_terminate_destroy(m_mpv);
    }
}

void PlayerWidget::play(
    const char* filename
)
{
    if (!m_mpv)
    {
        return;
    }

    const char* command[] =
    {
        "loadfile",
        filename,
        "replace",
        nullptr
    };

    const int result = mpv_command_async(m_mpv, 0, command);
    if (result < 0) {
        const QString message = QStringLiteral("MPV load failed: %1").arg(QString::fromUtf8(mpv_error_string(result)));
        qWarning() << message << filename;
        emit mediaFailed(QString::fromUtf8(filename), message);
    }
}

void PlayerWidget::stop()
{
    if (!m_mpv)
    {
        return;
    }

    const char* command[] =
    {
        "stop",
        nullptr
    };

    const int result = mpv_command_async(m_mpv, 0, command);
    if (result < 0) {
        qWarning() << "MPV stop failed:" << mpv_error_string(result);
    }
}

void PlayerWidget::loadMedia(const QString &path, int startSeconds)
{
    if (!m_mpv) {
        emit mediaFailed(path, QStringLiteral("MPV is not available."));
        return;
    }
    m_waitingForPlaybackRestart = true;
    m_currentPath = path;
    m_lastPlaybackErrorDetail.clear();
    m_pendingStartSeconds = startSeconds;
    emit mediaStatus(QStringLiteral("Loading stream %1").arg(path));

    const QByteArray pathBytes = path.toUtf8();

    const char* command[] = {
        "loadfile",
        pathBytes.constData(),
        "replace",
        nullptr
    };
    const int result = mpv_command_async(m_mpv, 0, command);
    if (result < 0) {
        const QString message = QStringLiteral("MPV load failed for %1: %2")
            .arg(path, QString::fromUtf8(mpv_error_string(result)));
        m_waitingForPlaybackRestart = false;
        emit mediaStatus(message);
        emit mediaFailed(path, message);
        qWarning() << message;
    }
}

void PlayerWidget::applyAudioDefaults()
{
    if (!m_mpv) {
        return;
    }

    // Re-apply the viewer's volume and mute on every file, so they carry
    // across channel changes instead of resetting.
    const char *audio = "auto";
    const char *aid = "auto";
    int muted = m_muted ? 1 : 0;
    double volume = m_volume;
    mpv_set_property(m_mpv, "audio", MPV_FORMAT_STRING, &audio);
    mpv_set_property(m_mpv, "aid", MPV_FORMAT_STRING, &aid);
    mpv_set_property(m_mpv, "mute", MPV_FORMAT_FLAG, &muted);
    mpv_set_property(m_mpv, "volume", MPV_FORMAT_DOUBLE, &volume);
}

void PlayerWidget::adjustVolume(int delta)
{
    if (!m_mpv) {
        return;
    }

    m_volume = std::clamp(m_volume + delta, 0, 100);
    m_muted = false;
    // The osd-msg-bar prefix makes mpv draw its own volume bar over the video.
    const QByteArray volume = QByteArray::number(m_volume);
    const char *unmute[] = {"set", "mute", "no", nullptr};
    mpv_command_async(m_mpv, 0, unmute);
    const char *command[] = {"osd-msg-bar", "set", "volume", volume.constData(), nullptr};
    mpv_command_async(m_mpv, 0, command);
}

void PlayerWidget::toggleMute()
{
    if (!m_mpv) {
        return;
    }

    m_muted = !m_muted;
    const char *command[] = {"osd-msg", "set", "mute", m_muted ? "yes" : "no", nullptr};
    mpv_command_async(m_mpv, 0, command);
}

void PlayerWidget::reportAudioState()
{
    if (!m_mpv) {
        return;
    }

    double volume = 0.0;
    int64_t audioTrack = 0;
    int64_t audioDeviceCount = 0;
    int64_t trackCount = 0;
    int muted = 0;

    const int volumeResult = mpv_get_property(m_mpv, "volume", MPV_FORMAT_DOUBLE, &volume);
    const int muteResult = mpv_get_property(m_mpv, "mute", MPV_FORMAT_FLAG, &muted);
    const int aidResult = mpv_get_property(m_mpv, "aid", MPV_FORMAT_INT64, &audioTrack);
    const int deviceResult = mpv_get_property(m_mpv, "audio-device-list/count", MPV_FORMAT_INT64, &audioDeviceCount);
    const int trackResult = mpv_get_property(m_mpv, "track-list/count", MPV_FORMAT_INT64, &trackCount);

    const QString message = QStringLiteral("MPV audio state: volume=%1 mute=%2 aid=%3 devices=%4 tracks=%5")
        .arg(volumeResult < 0 ? QStringLiteral("?") : QString::number(volume, 'f', 0),
             muteResult < 0 ? QStringLiteral("?") : QString::number(muted),
             aidResult < 0 ? QStringLiteral("?") : QString::number(audioTrack),
             deviceResult < 0 ? QStringLiteral("?") : QString::number(audioDeviceCount),
             trackResult < 0 ? QStringLiteral("?") : QString::number(trackCount));
    emit mediaStatus(message);
    qWarning() << message;
}

void PlayerWidget::reportPlaybackStats()
{
    if (!m_mpv) {
        return;
    }

    const auto property = [this](const char *name) {
        char *value = mpv_get_property_string(m_mpv, name);
        if (!value) {
            return QStringLiteral("?");
        }
        const QString result = QString::fromUtf8(value);
        mpv_free(value);
        return result;
    };

    // Stay quiet while playback is healthy: only log when frames were dropped
    // since the last check or the open file count climbs (a descriptor leak).
    int64_t dropped = 0;
    int64_t decoderDropped = 0;
    mpv_get_property(m_mpv, "frame-drop-count", MPV_FORMAT_INT64, &dropped);
    mpv_get_property(m_mpv, "decoder-frame-drop-count", MPV_FORMAT_INT64, &decoderDropped);
    const int openFiles = QDir(QStringLiteral("/proc/self/fd")).entryList(QDir::AllEntries | QDir::System | QDir::NoDotAndDotDot).size();
    const bool framesDropped = dropped > m_lastDroppedFrames || decoderDropped > m_lastDecoderDroppedFrames;
    const bool openFilesGrew = m_openFilesHighWater > 0 && openFiles > m_openFilesHighWater + 32;
    m_lastDroppedFrames = dropped;
    m_lastDecoderDroppedFrames = decoderDropped;
    if (m_openFilesHighWater == 0 || openFilesGrew) {
        m_openFilesHighWater = openFiles;
    }
    if (!framesDropped && !openFilesGrew) {
        return;
    }

    const qreal scale = devicePixelRatioF();
    const QString message = QStringLiteral("MPV playback stats: codec=%1 hwdec=%2 video=%3x%4 fps=%5 output=%6x%7 "
                                           "dropped=%8 decoder-dropped=%9 delayed=%10 avsync=%11 cache=%12s fds=%13 [%14]")
        .arg(property("video-codec"),
             property("hwdec-current"),
             property("video-params/w"),
             property("video-params/h"),
             property("container-fps"),
             QString::number(qRound(width() * scale)),
             QString::number(qRound(height() * scale)),
             property("frame-drop-count"),
             property("decoder-frame-drop-count"))
        .arg(property("vo-delayed-frame-count"),
             property("avsync"),
             property("demuxer-cache-duration"),
             QString::number(openFiles),
             openFileSummary());
    qWarning() << message;
}

void PlayerWidget::handleMpvEvents()
{
    if (!m_mpv)
    {
        return;
    }

    while (true)
    {
        mpv_event *event = mpv_wait_event(m_mpv, 0);
        if (!event || event->event_id == MPV_EVENT_NONE)
        {
            break;
        }

        if (event->event_id == MPV_EVENT_FILE_LOADED) {
            applyAudioDefaults();
            const bool isHLS = m_currentPath.contains(QStringLiteral(".m3u8"), Qt::CaseInsensitive);
            if (m_pendingStartSeconds > 0 && !isHLS) {
                double position = static_cast<double>(m_pendingStartSeconds);
                const int seekResult = mpv_set_property(m_mpv, "time-pos", MPV_FORMAT_DOUBLE, &position);
                if (seekResult < 0) {
                    const QString message = QStringLiteral("MPV seek failed for %1: %2")
                        .arg(m_currentPath, QString::fromUtf8(mpv_error_string(seekResult)));
                    emit mediaStatus(message);
                    emit mediaFailed(m_currentPath, message);
                    qWarning() << message;
                } else {
                    emit mediaStatus(QStringLiteral("Loaded %1 at %2s").arg(m_currentPath).arg(m_pendingStartSeconds));
                }
            } else {
                emit mediaStatus(QStringLiteral("Loaded %1 %2").arg(isHLS ? QStringLiteral("HLS stream") : QStringLiteral("media")).arg(m_currentPath));
            }
            reportAudioState();
            m_pendingStartSeconds = 0;
            continue;
        }

        if (event->event_id == MPV_EVENT_PLAYBACK_RESTART) {
            if (m_waitingForPlaybackRestart) {
                m_waitingForPlaybackRestart = false;
                emit mediaStatus(QStringLiteral("Playing %1").arg(m_currentPath));
                emit mediaLoaded(m_currentPath);
            }
            // mpv's drop counters restart with each file.
            m_lastDroppedFrames = 0;
            m_lastDecoderDroppedFrames = 0;
            m_statsTimer.start();
            continue;
        }

        if (event->event_id == MPV_EVENT_END_FILE) {
            m_statsTimer.stop();
            auto *endEvent = static_cast<mpv_event_end_file *>(event->data);
            if (endEvent && endEvent->reason == MPV_END_FILE_REASON_ERROR) {
                const QString errorText = endEvent->error < 0
                    ? QString::fromUtf8(mpv_error_string(endEvent->error))
                    : QStringLiteral("unknown mpv error");
                QString message = QStringLiteral("Playback failed for %1: %2").arg(m_currentPath, errorText);
                if (!m_lastPlaybackErrorDetail.isEmpty() && !message.contains(m_lastPlaybackErrorDetail)) {
                    message += QStringLiteral(" (%1)").arg(m_lastPlaybackErrorDetail);
                }
                emit mediaStatus(message);
                emit mediaFailed(m_currentPath, message);
                qWarning() << message;
            } else if (endEvent && endEvent->reason == MPV_END_FILE_REASON_EOF) {
                emit mediaStatus(QStringLiteral("Media ended %1").arg(m_currentPath));
                emit mediaEnded(m_currentPath);
            }
            continue;
        }

        if (event->event_id == MPV_EVENT_LOG_MESSAGE) {
            auto *logMessage = static_cast<mpv_event_log_message *>(event->data);
            if (logMessage && logMessage->text) {
                const QString level = QString::fromUtf8(logMessage->level ? logMessage->level : "");
                const QString detail = QString::fromUtf8(logMessage->text).trimmed();
                if (level == QStringLiteral("error") || level == QStringLiteral("fatal")) {
                    m_lastPlaybackErrorDetail = detail;
                }
                const QString message = QStringLiteral("MPV %1: %2")
                    .arg(QString::fromUtf8(logMessage->prefix ? logMessage->prefix : "log"),
                         detail);
                emit mediaStatus(message);
                qWarning() << message;
            }
        }
    }
}
