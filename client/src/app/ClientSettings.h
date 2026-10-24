#pragma once

#include <QString>

struct ClientSettings
{
    QString backendURL = QStringLiteral("http://127.0.0.1:8080");
    static ClientSettings load();
    void save() const;

    // The last backend that served channels, tried when discovery finds
    // nothing so a missed mDNS reply does not strand the appliance.
    static QString lastWorkingServerURL();
    static void rememberWorkingServer(const QString &url);

    // Guide text size chosen on the settings page, or fallback if never set.
    static int guideTextSize(int fallback);
    static void setGuideTextSize(int textSize);
};
