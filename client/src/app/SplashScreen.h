#pragma once

#include <QPixmap>

class QSize;

class SplashScreen
{
public:
    static QPixmap pixmap(const QSize &size);
};
