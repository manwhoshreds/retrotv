#include "ChannelTransitionOverlay.h"

#include <QApplication>
#include <QCoreApplication>
#include <QDir>
#include <QFile>
#include <QFrame>
#include <QGraphicsOpacityEffect>
#include <QHBoxLayout>
#include <QLabel>
#include <QMoveEvent>
#include <QPainter>
#include <QPaintEvent>
#include <QPauseAnimation>
#include <QPropertyAnimation>
#include <QResizeEvent>
#include <QSequentialAnimationGroup>
#include <QStringList>
#include <QTimer>
#include <QVBoxLayout>

namespace {

QString channelNumberText(int channelNumber)
{
    return channelNumber > 0
               ? QStringLiteral("%1").arg(channelNumber, 3, 10, QLatin1Char('0'))
               : QStringLiteral("---");
}

QString loadStyleSheet()
{
    const QString appDir = QCoreApplication::applicationDirPath();
    const QString cwd = QDir::currentPath();
    const QStringList candidates = {
        cwd + QStringLiteral("/styles/channel-transition.qss"),
        cwd + QStringLiteral("/../styles/channel-transition.qss"),
        appDir + QStringLiteral("/styles/channel-transition.qss"),
        appDir + QStringLiteral("/../styles/channel-transition.qss"),
    };

    for (const QString &path : candidates) {
        QFile file(path);
        if (!file.open(QIODevice::ReadOnly | QIODevice::Text)) {
            continue;
        }
        return QString::fromUtf8(file.readAll());
    }
    return {};
}

QLabel *label(const QString &text, const QString &objectName)
{
    auto *value = new QLabel(text);
    value->setObjectName(objectName);
    value->setAlignment(Qt::AlignCenter);
    value->setWordWrap(true);
    return value;
}

}

ChannelTransitionOverlay::ChannelTransitionOverlay(QWidget *parent)
    : QWidget(parent)
{
    setAttribute(Qt::WA_TransparentForMouseEvents);
    setAttribute(Qt::WA_NoSystemBackground, false);
    setFocusPolicy(Qt::NoFocus);
    hide();

    m_panel = new QFrame(this);
    m_panel->setObjectName(QStringLiteral("transitionPanel"));
    m_panel->setAttribute(Qt::WA_TransparentForMouseEvents);
    m_panel->setMinimumWidth(430);
    m_panel->setMaximumWidth(560);
    m_panel->hide();

    auto *panelLayout = new QHBoxLayout(m_panel);
    panelLayout->setContentsMargins(20, 14, 22, 14);
    panelLayout->setSpacing(18);

    auto *numberBlock = new QVBoxLayout();
    numberBlock->setContentsMargins(0, 0, 0, 0);
    numberBlock->setSpacing(0);
    numberBlock->addWidget(label(QStringLiteral("CH"), QStringLiteral("transitionEyebrow")));
    m_numberLabel = label(QStringLiteral("---"), QStringLiteral("transitionNumber"));
    numberBlock->addWidget(m_numberLabel);
    panelLayout->addLayout(numberBlock);

    auto *rule = new QFrame(m_panel);
    rule->setObjectName(QStringLiteral("transitionRule"));
    rule->setFrameShape(QFrame::VLine);
    panelLayout->addWidget(rule);

    auto *detailLayout = new QVBoxLayout();
    detailLayout->setContentsMargins(0, 0, 0, 0);
    detailLayout->setSpacing(4);
    m_channelNameLabel = label(QStringLiteral("RETROTV"), QStringLiteral("transitionChannel"));
    m_channelNameLabel->setAlignment(Qt::AlignLeft | Qt::AlignVCenter);
    detailLayout->addWidget(m_channelNameLabel);
    m_titleLabel = label(QString(), QStringLiteral("transitionTitle"));
    m_titleLabel->setAlignment(Qt::AlignLeft | Qt::AlignVCenter);
    detailLayout->addWidget(m_titleLabel);
    m_timeLabel = label(QString(), QStringLiteral("transitionTime"));
    m_timeLabel->setAlignment(Qt::AlignLeft | Qt::AlignVCenter);
    detailLayout->addWidget(m_timeLabel);
    panelLayout->addLayout(detailLayout, 1);

    m_opacityEffect = new QGraphicsOpacityEffect(this);
    m_opacityEffect->setOpacity(1.0);
    m_panel->setGraphicsEffect(m_opacityEffect);

    auto *hold = new QPauseAnimation(5000, this);

    auto *fadeOut = new QPropertyAnimation(m_opacityEffect, "opacity", this);
    fadeOut->setDuration(180);
    fadeOut->setStartValue(1.0);
    fadeOut->setEndValue(0.0);
    fadeOut->setEasingCurve(QEasingCurve::InCubic);

    m_animation = new QSequentialAnimationGroup(this);
    m_animation->addAnimation(hold);
    m_animation->addAnimation(fadeOut);
    connect(m_animation, &QSequentialAnimationGroup::finished, this, [this]() {
        m_panel->hide();
    });

    applyStyleSheet();
}

ChannelTransitionOverlay::~ChannelTransitionOverlay()
{
    if (m_panel) {
        m_panel->setGraphicsEffect(nullptr);
        delete m_panel;
        m_panel = nullptr;
    }
}

void ChannelTransitionOverlay::startTransition(const ChannelTransitionInfo &info)
{
    ++m_transitionGeneration;
    m_numberEntryActive = false;
    const int generation = m_transitionGeneration;
    m_showBlackout = true;
    m_minimumBlackoutElapsed = false;
    m_blackoutClearRequested = false;
    update();

    updateTransitionInfo(info);

    if (m_animation->state() == QAbstractAnimation::Running) {
        m_animation->stop();
    }
    m_opacityEffect->setOpacity(1.0);
    show();
    raise();
    positionPanel();
    m_panel->show();
    m_panel->raise();

    QTimer::singleShot(120, this, [this, generation]() {
        if (generation != m_transitionGeneration) {
            return;
        }
        m_minimumBlackoutElapsed = true;
        if (m_blackoutClearRequested) {
            clearBlackoutWhenReady();
        }
    });
}

void ChannelTransitionOverlay::showNumberEntry(const QString &digits)
{
    ++m_transitionGeneration;
    m_numberEntryActive = true;
    m_showBlackout = false;
    update();
    if (m_animation->state() == QAbstractAnimation::Running) {
        m_animation->stop();
    }
    m_opacityEffect->setOpacity(1.0);
    updateNumberEntry(digits);
    // The panel is a child of this overlay, so the overlay itself must be
    // visible (transparent, since the blackout is off) for digits to show.
    show();
    raise();
    positionPanel();
    m_panel->show();
    m_panel->raise();
}

void ChannelTransitionOverlay::updateNumberEntry(const QString &digits)
{
    if (!m_numberEntryActive) {
        showNumberEntry(digits);
        return;
    }
    m_numberLabel->setText(digits.isEmpty() ? QStringLiteral("---") : digits);
    m_channelNameLabel->setText(QStringLiteral("ENTER CHANNEL"));
    m_titleLabel->hide();
    m_timeLabel->hide();
    positionPanel();
}

void ChannelTransitionOverlay::hideNumberEntry()
{
    if (!m_numberEntryActive) {
        return;
    }
    m_numberEntryActive = false;
    m_panel->hide();
    hide();
}

void ChannelTransitionOverlay::updateTransitionInfo(const ChannelTransitionInfo &info)
{
    m_numberLabel->setText(channelNumberText(info.channelNumber));
    m_channelNameLabel->setText(info.channelName.isEmpty() ? QStringLiteral("RETROTV") : info.channelName.toUpper());
    m_titleLabel->setText(info.title);
    m_titleLabel->setVisible(!info.title.isEmpty());
    m_timeLabel->setText(info.timeRange);
    m_timeLabel->setVisible(!info.timeRange.isEmpty());
    positionPanel();
}

void ChannelTransitionOverlay::clearBlackoutWhenReady()
{
    m_blackoutClearRequested = true;
    if (!m_minimumBlackoutElapsed) {
        return;
    }

    const int generation = m_transitionGeneration;
    QTimer::singleShot(0, this, [this, generation]() {
        if (generation != m_transitionGeneration) {
            return;
        }
        m_showBlackout = false;
        update();
        hide();
        if (m_animation->state() == QAbstractAnimation::Running) {
            m_animation->stop();
        }
        m_opacityEffect->setOpacity(1.0);
        positionPanel();
        m_panel->show();
        m_panel->raise();
        m_animation->start();
    });
}

void ChannelTransitionOverlay::moveEvent(QMoveEvent *event)
{
    QWidget::moveEvent(event);
    positionPanel();
}

void ChannelTransitionOverlay::paintEvent(QPaintEvent *event)
{
    Q_UNUSED(event);

    if (!m_showBlackout) {
        return;
    }

    QPainter painter(this);
    painter.fillRect(rect(), QColor(0, 0, 0, 255));
}

void ChannelTransitionOverlay::resizeEvent(QResizeEvent *event)
{
    QWidget::resizeEvent(event);
    positionPanel();
}

void ChannelTransitionOverlay::applyStyleSheet()
{
    const QString styleSheet = loadStyleSheet();
    if (!styleSheet.isEmpty()) {
        setStyleSheet(styleSheet);
        if (m_panel) {
            m_panel->setStyleSheet(styleSheet);
        }
    }
}

void ChannelTransitionOverlay::positionPanel()
{
    if (!m_panel) {
        return;
    }

    m_panel->adjustSize();
    m_panel->move(QPoint(30, 30));
}
