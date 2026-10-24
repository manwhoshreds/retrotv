#include "ServerDiscovery.h"

#include <QElapsedTimer>
#include <QHostAddress>
#include <QNetworkDatagram>
#include <QUdpSocket>

namespace {
quint16 word(const QByteArray &data, int offset)
{
    return (quint8(data[offset]) << 8) | quint8(data[offset + 1]);
}

QString dnsName(const QByteArray &data, int &offset, int depth = 0)
{
    if (depth > 12) return {};
    QStringList labels;
    int cursor = offset;
    bool jumped = false;
    while (cursor < data.size()) {
        const quint8 length = quint8(data[cursor++]);
        if (length == 0) break;
        if ((length & 0xc0) == 0xc0) {
            if (cursor >= data.size()) return {};
            int pointer = ((length & 0x3f) << 8) | quint8(data[cursor++]);
            int pointed = pointer;
            labels << dnsName(data, pointed, depth + 1);
            jumped = true;
            break;
        }
        if (cursor + length > data.size()) return {};
        labels << QString::fromUtf8(data.constData() + cursor, length);
        cursor += length;
    }
    Q_UNUSED(jumped);
    offset = cursor;
    return labels.join('.');
}

QByteArray queryPacket()
{
    QByteArray packet(12, '\0');
    packet[5] = 1;
    for (const QByteArray &label : {QByteArray("_retrotv"), QByteArray("_tcp"), QByteArray("local")}) {
        packet.append(char(label.size())); packet.append(label);
    }
    packet.append('\0'); packet.append('\0'); packet.append('\x0c');
    packet.append('\0'); packet.append('\x01');
    return packet;
}

QList<DiscoveredServer> parseResponse(const QByteArray &data)
{
    QList<DiscoveredServer> found;
    if (data.size() < 12) return found;
    int offset = 12;
    const int questions = word(data, 4);
    const int records = word(data, 6) + word(data, 8) + word(data, 10);
    for (int i = 0; i < questions; ++i) {
        dnsName(data, offset); offset += 4;
    }
    for (int i = 0; i < records && offset + 10 <= data.size(); ++i) {
        const QString owner = dnsName(data, offset);
        if (offset + 10 > data.size()) break;
        const quint16 type = word(data, offset); offset += 8;
        const quint16 length = word(data, offset); offset += 2;
        const int end = offset + length;
        if (end > data.size()) break;
        if (type == 33 && length >= 7) {
            const quint16 port = word(data, offset + 4);
            int targetOffset = offset + 6;
            QString target = dnsName(data, targetOffset);
            if (target.endsWith('.')) target.chop(1);
            if (!target.isEmpty() && port > 0) {
                QString instance = owner.section("._retrotv", 0, 0);
                found.append({instance.isEmpty() ? QStringLiteral("RetroTV") : instance,
                    QStringLiteral("http://%1:%2").arg(target).arg(port)});
            }
        }
        offset = end;
    }
    return found;
}
}

QList<DiscoveredServer> discoverRetroTVServers(int timeoutMilliseconds)
{
    QList<DiscoveredServer> found;
    QUdpSocket socket;
    if (!socket.bind(QHostAddress::AnyIPv4, 5353, QUdpSocket::ShareAddress | QUdpSocket::ReuseAddressHint)) return found;
    socket.joinMulticastGroup(QHostAddress(QStringLiteral("224.0.0.251")));
    // Multicast is easily lost (especially on Wi-Fi), so repeat the query
    // until a server answers or time runs out.
    constexpr qint64 kQueryIntervalMilliseconds = 600;
    qint64 lastQuery = -kQueryIntervalMilliseconds;
    QElapsedTimer timer; timer.start();
    // Once a server answers, give others a short grace period, then stop.
    constexpr qint64 kGraceMilliseconds = 300;
    qint64 firstFound = -1;
    while (timer.elapsed() < timeoutMilliseconds) {
        if (firstFound < 0 && !found.isEmpty()) {
            firstFound = timer.elapsed();
        }
        if (firstFound >= 0 && timer.elapsed() - firstFound >= kGraceMilliseconds) {
            break;
        }
        if (found.isEmpty() && timer.elapsed() - lastQuery >= kQueryIntervalMilliseconds) {
            socket.writeDatagram(queryPacket(), QHostAddress(QStringLiteral("224.0.0.251")), 5353);
            lastQuery = timer.elapsed();
        }
        if (!socket.waitForReadyRead(qMin(200, timeoutMilliseconds - int(timer.elapsed())))) continue;
        while (socket.hasPendingDatagrams()) {
            for (const DiscoveredServer &server : parseResponse(socket.receiveDatagram().data())) {
                bool duplicate = false;
                for (const DiscoveredServer &existing : found) duplicate = duplicate || existing.url == server.url;
                if (!duplicate) found.append(server);
            }
        }
    }
    return found;
}
