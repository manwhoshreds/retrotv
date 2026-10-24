#pragma once

#include <QString>
#include <QList>

struct DiscoveredServer {
    QString name;
    QString url;
};

QList<DiscoveredServer> discoverRetroTVServers(int timeoutMilliseconds = 2500);
