#pragma once

#include <QWidget>

class QLabel;
class QFrame;
class QGraphicsOpacityEffect;
class QMoveEvent;
class QPropertyAnimation;
class QResizeEvent;
class QSequentialAnimationGroup;

struct ChannelTransitionInfo
{
    int channelNumber = 0;
    QString channelName;
    QString title;
    QString timeRange;
};

class ChannelTransitionOverlay : public QWidget
{
    Q_OBJECT

public:
    explicit ChannelTransitionOverlay(QWidget *parent = nullptr);
    ~ChannelTransitionOverlay() override;
    void startTransition(const ChannelTransitionInfo &info);
    void updateTransitionInfo(const ChannelTransitionInfo &info);
    void clearBlackoutWhenReady();
    void showNumberEntry(const QString &digits);
    void updateNumberEntry(const QString &digits);
    void hideNumberEntry();

protected:
    void moveEvent(QMoveEvent *event) override;
    void paintEvent(QPaintEvent *event) override;
    void resizeEvent(QResizeEvent *event) override;

private:
    void applyStyleSheet();
    void positionPanel();

    QLabel *m_numberLabel = nullptr;
    QLabel *m_channelNameLabel = nullptr;
    QLabel *m_titleLabel = nullptr;
    QLabel *m_timeLabel = nullptr;
    QFrame *m_panel = nullptr;
    QGraphicsOpacityEffect *m_opacityEffect = nullptr;
    QSequentialAnimationGroup *m_animation = nullptr;
    bool m_showBlackout = false;
    bool m_minimumBlackoutElapsed = false;
    bool m_blackoutClearRequested = false;
    bool m_numberEntryActive = false;
    int m_transitionGeneration = 0;
};
