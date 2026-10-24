#include "ClientSettings.h"

#include <QSettings>

ClientSettings ClientSettings::load()
{
    QSettings settings;
    ClientSettings result;
    result.backendURL = settings.value(QStringLiteral("connection/backendURL"), result.backendURL).toString().trimmed();
    if (result.backendURL.isEmpty()) {
        result.backendURL = QStringLiteral("http://127.0.0.1:8080");
    }
    while (result.backendURL.endsWith(QLatin1Char('/'))) {
        result.backendURL.chop(1);
    }
    return result;
}

void ClientSettings::save() const
{
    QSettings settings;
    settings.setValue(QStringLiteral("connection/backendURL"), backendURL);
    settings.sync();
}

QString ClientSettings::lastWorkingServerURL()
{
    return QSettings().value(QStringLiteral("connection/lastWorkingServerURL")).toString().trimmed();
}

void ClientSettings::rememberWorkingServer(const QString &url)
{
    QSettings settings;
    if (settings.value(QStringLiteral("connection/lastWorkingServerURL")).toString() == url) {
        return;
    }
    settings.setValue(QStringLiteral("connection/lastWorkingServerURL"), url);
    settings.sync();
}

int ClientSettings::guideTextSize(int fallback)
{
    bool ok = false;
    const int value = QSettings().value(QStringLiteral("guide/textSize")).toInt(&ok);
    return ok ? value : fallback;
}

void ClientSettings::setGuideTextSize(int textSize)
{
    QSettings settings;
    settings.setValue(QStringLiteral("guide/textSize"), textSize);
    settings.sync();
}
