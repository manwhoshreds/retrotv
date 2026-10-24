#include "SplashScreen.h"

#include <algorithm>
#include <cmath>

#include <QColor>
#include <QFont>
#include <QLinearGradient>
#include <QPainter>
#include <QPainterPath>
#include <QPen>
#include <QPointF>
#include <QRadialGradient>
#include <QRectF>
#include <QSize>
#include <QVector>

namespace {

QFont font(const QString &family, int pointSize, int weight = QFont::Normal, bool italic = false)
{
    QFont result(family, pointSize, weight, italic);
    result.setStyleHint(QFont::SansSerif);
    return result;
}

void drawStars(QPainter &painter, const QRectF &rect)
{
    painter.setPen(QPen(QColor(210, 225, 255, 210), 1.4));
    for (int index = 0; index < 85; ++index) {
        const int x = static_cast<int>(rect.left()) + ((index * 97) % static_cast<int>(rect.width()));
        const int y = static_cast<int>(rect.top()) + 22 + ((index * 53) % static_cast<int>(rect.height() * 0.48));
        const int size = index % 13 == 0 ? 3 : 1;
        painter.drawLine(QPointF(x - size, y), QPointF(x + size, y));
        painter.drawLine(QPointF(x, y - size), QPointF(x, y + size));
    }
}

void drawGrid(QPainter &painter, const QRectF &screen)
{
    const qreal horizon = screen.top() + screen.height() * 0.55;
    const QPointF vanishing(screen.center().x(), horizon - 12);
    QPen blue(QColor(0, 117, 255, 175), 2);
    QPen magenta(QColor(255, 45, 210, 175), 2);

    for (int index = -12; index <= 12; ++index) {
        const qreal bottomX = screen.center().x() + index * screen.width() * 0.095;
        painter.setPen(index < 0 ? blue : magenta);
        painter.drawLine(QPointF(bottomX, screen.bottom() - 42), vanishing);
    }

    for (int index = 0; index < 13; ++index) {
        const qreal t = index / 12.0;
        const qreal y = horizon + std::pow(t, 1.85) * (screen.bottom() - 42 - horizon);
        painter.setPen(index % 2 == 0 ? blue : magenta);
        painter.drawLine(QPointF(screen.left() + 38, y), QPointF(screen.right() - 38, y));
    }
}

void drawSunAndMountains(QPainter &painter, const QRectF &screen)
{
    const QPointF center(screen.center().x(), screen.top() + screen.height() * 0.45);
    const qreal radius = screen.width() * 0.235;
    QPainterPath sun;
    sun.addEllipse(center, radius, radius);
    painter.save();
    painter.setClipPath(sun);
    QLinearGradient sunset(center.x(), center.y() - radius, center.x(), center.y() + radius);
    sunset.setColorAt(0, QColor(255, 197, 42));
    sunset.setColorAt(0.52, QColor(255, 66, 112));
    sunset.setColorAt(1, QColor(203, 24, 180));
    painter.fillPath(sun, sunset);

    painter.setPen(QPen(QColor(8, 12, 31, 130), 10));
    for (int stripe = -2; stripe < 4; ++stripe) {
        const qreal y = center.y() + stripe * radius * 0.27;
        painter.drawLine(QPointF(center.x() - radius, y), QPointF(center.x() + radius, y));
    }
    painter.restore();

    QPainterPath mountains;
    const qreal base = screen.top() + screen.height() * 0.55;
    mountains.moveTo(screen.left() + 34, base);
    const int peaks = 18;
    for (int index = 0; index <= peaks; ++index) {
        const qreal x = screen.left() + 34 + (screen.width() - 68) * index / peaks;
        const qreal y = base - ((index * 37) % 55) - (index % 4 == 0 ? 32 : 0);
        mountains.lineTo(x, y);
    }
    mountains.lineTo(screen.right() - 34, base + 20);
    mountains.lineTo(screen.left() + 34, base + 20);
    mountains.closeSubpath();
    painter.fillPath(mountains, QColor(6, 8, 20));
}

void drawTv(QPainter &painter, const QRectF &screen)
{
    const QRectF tv(screen.center().x() - 210, screen.top() + screen.height() * 0.2, 420, 245);

    painter.setPen(QPen(QColor(178, 205, 220), 4));
    painter.drawLine(tv.center() + QPointF(-48, 0), tv.topLeft() + QPointF(65, -128));
    painter.drawLine(tv.center() + QPointF(48, 0), tv.topRight() + QPointF(-65, -128));
    painter.setBrush(QColor(16, 18, 22));
    painter.drawEllipse(tv.center() + QPointF(-48, 0), 17, 11);
    painter.drawEllipse(tv.center() + QPointF(48, 0), 17, 11);

    QPainterPath body;
    body.addRoundedRect(tv, 18, 18);
    painter.fillPath(body, QColor(14, 16, 19));
    painter.setPen(QPen(QColor(210, 220, 225), 3));
    painter.drawPath(body);

    const QRectF tube(tv.left() + 28, tv.top() + 28, tv.width() * 0.68, tv.height() - 58);
    QPainterPath tubePath;
    tubePath.addRoundedRect(tube, 22, 22);
    painter.fillPath(tubePath, QColor(5, 8, 12));
    painter.setPen(QPen(QColor(70, 78, 88), 3));
    painter.drawPath(tubePath);

    const QVector<QColor> bars{
        QColor(245, 245, 245), QColor(255, 226, 32), QColor(22, 215, 227),
        QColor(12, 206, 96), QColor(218, 43, 202), QColor(238, 18, 43), QColor(0, 47, 184),
    };
    const qreal barWidth = tube.width() / bars.size();
    painter.save();
    painter.setClipPath(tubePath);
    for (int index = 0; index < bars.size(); ++index) {
        painter.fillRect(QRectF(tube.left() + index * barWidth, tube.top(), barWidth + 1, tube.height()), bars[index]);
    }
    painter.fillRect(tube, QColor(255, 255, 255, 24));
    painter.restore();

    const QRectF panel(tv.right() - 88, tv.top() + 28, 62, tv.height() - 58);
    painter.fillRect(panel, QColor(8, 9, 11));
    painter.setPen(QPen(QColor(80, 86, 92), 2));
    painter.drawRoundedRect(panel, 8, 8);
    painter.setBrush(QColor(25, 25, 28));
    painter.drawEllipse(panel.center().x() - 23, panel.top() + 18, 46, 46);
    painter.drawEllipse(panel.center().x() - 23, panel.top() + 82, 46, 46);
    painter.drawRect(QRectF(panel.left() + 10, panel.bottom() - 58, panel.width() - 20, 42));
}

void drawLogo(QPainter &painter, const QRectF &screen)
{
    const QRectF logo(screen.left() + screen.width() * 0.18, screen.top() + screen.height() * 0.55,
                      screen.width() * 0.64, screen.height() * 0.16);
    QFont retro = font(QStringLiteral("Arial Black"), static_cast<int>(screen.height() * 0.106), QFont::Black);
    painter.setFont(retro);

    QPen shadow(QColor(0, 190, 255, 170), 7);
    painter.setPen(shadow);
    painter.drawText(logo.translated(5, 5), Qt::AlignLeft | Qt::AlignVCenter, QStringLiteral("RETRO"));

    QLinearGradient chrome(logo.left(), logo.top(), logo.left(), logo.bottom());
    chrome.setColorAt(0, QColor(162, 244, 255));
    chrome.setColorAt(0.45, QColor(255, 255, 255));
    chrome.setColorAt(0.7, QColor(64, 91, 255));
    chrome.setColorAt(1, QColor(255, 255, 255));
    painter.setPen(QPen(QBrush(chrome), 4));
    painter.drawText(logo, Qt::AlignLeft | Qt::AlignVCenter, QStringLiteral("RETRO"));

    QFont script = font(QStringLiteral("Arial Black"), static_cast<int>(screen.height() * 0.126), QFont::Black, true);
    painter.setFont(script);
    painter.setPen(QPen(QColor(255, 32, 178), 8));
    painter.drawText(logo.translated(screen.width() * 0.47, 6), Qt::AlignLeft | Qt::AlignVCenter, QStringLiteral("TV"));
    painter.setPen(QPen(QColor(255, 94, 211), 3));
    painter.drawText(logo.translated(screen.width() * 0.47, 6), Qt::AlignLeft | Qt::AlignVCenter, QStringLiteral("TV"));
}

void drawLoading(QPainter &painter, const QRectF &screen)
{
    const QRectF tagline(screen.left(), screen.top() + screen.height() * 0.72, screen.width(), 45);
    painter.setFont(font(QStringLiteral("Monospace"), static_cast<int>(screen.height() * 0.026), QFont::DemiBold));
    painter.setPen(QColor(235, 244, 255));
    painter.drawText(tagline, Qt::AlignCenter, QStringLiteral("YOUR CHANNELS. YOUR MOVIES. YOUR TV."));

    const QRectF box(screen.center().x() - 310, screen.top() + screen.height() * 0.81, 620, 84);
    QPainterPath path;
    path.addRoundedRect(box, 12, 12);
    painter.fillPath(path, QColor(0, 0, 0, 190));
    painter.setPen(QPen(QColor(255, 37, 192), 3));
    painter.drawPath(path);

    painter.setFont(font(QStringLiteral("Monospace"), static_cast<int>(screen.height() * 0.029), QFont::DemiBold));
    painter.setPen(QColor(237, 250, 255));
    painter.drawText(box.adjusted(40, 0, 0, 0), Qt::AlignVCenter | Qt::AlignLeft, QStringLiteral("TUNING IN..."));

    for (int index = 0; index < 12; ++index) {
        const QColor color = QColor::fromHsv(188 + index * 7, 220, 245);
        painter.fillRect(QRectF(box.left() + 365 + index * 14, box.center().y() - 15, 8, 30), color);
    }
}

}

QPixmap SplashScreen::pixmap(const QSize &size)
{
    const QSize target = size.isValid() ? size : QSize(1536, 1024);
    QPixmap pixmap(target);
    pixmap.fill(QColor(6, 7, 10));

    QPainter painter(&pixmap);
    painter.setRenderHint(QPainter::Antialiasing);
    painter.setRenderHint(QPainter::TextAntialiasing);

    QRectF frame(18, 18, target.width() - 36, target.height() - 36);
    QPainterPath framePath;
    framePath.addRoundedRect(frame, 44, 44);
    painter.fillPath(framePath, QColor(22, 20, 18));

    QRectF screen = frame.adjusted(34, 34, -34, -34);
    QPainterPath screenPath;
    screenPath.addRoundedRect(screen, 36, 36);
    painter.save();
    painter.setClipPath(screenPath);

    QLinearGradient sky(screen.left(), screen.top(), screen.left(), screen.bottom());
    sky.setColorAt(0, QColor(2, 5, 20));
    sky.setColorAt(0.48, QColor(14, 10, 55));
    sky.setColorAt(1, QColor(5, 6, 15));
    painter.fillPath(screenPath, sky);

    QRadialGradient glow(screen.center().x(), screen.top() + screen.height() * 0.43, screen.width() * 0.36);
    glow.setColorAt(0, QColor(18, 118, 255, 130));
    glow.setColorAt(0.48, QColor(196, 27, 255, 95));
    glow.setColorAt(1, QColor(0, 0, 0, 0));
    painter.fillRect(screen, glow);

    drawStars(painter, screen);
    drawSunAndMountains(painter, screen);
    drawGrid(painter, screen);
    drawTv(painter, screen);
    drawLogo(painter, screen);
    drawLoading(painter, screen);

    painter.setOpacity(0.12);
    painter.setPen(QPen(QColor(255, 255, 255), 1));
    for (int y = static_cast<int>(screen.top()); y < screen.bottom(); y += 4) {
        painter.drawLine(QPointF(screen.left(), y), QPointF(screen.right(), y));
    }
    painter.restore();

    painter.setOpacity(1);
    painter.setPen(QPen(QColor(55, 51, 47), 4));
    painter.drawPath(framePath);
    painter.setPen(QPen(QColor(0, 0, 0, 180), 22));
    painter.drawPath(screenPath);

    return pixmap;
}
