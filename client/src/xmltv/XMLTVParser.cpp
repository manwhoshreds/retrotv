#include "XMLTVParser.h"

#include <QByteArray>
#include <QFile>
#include <QString>
#include <QXmlStreamReader>

namespace {

std::string toStdString(const QString &value)
{
    return value.toStdString();
}

XMLTVChannel parseChannel(QXmlStreamReader &xml)
{
    XMLTVChannel channel;
    channel.id = toStdString(xml.attributes().value(QStringLiteral("id")).toString());

    while (!xml.atEnd()) {
        xml.readNext();

        if (xml.isStartElement() && xml.name() == QStringLiteral("display-name")) {
            channel.displayName = toStdString(xml.readElementText().trimmed());
            continue;
        }

        if (xml.isStartElement() && xml.name() == QStringLiteral("icon")) {
            channel.icon = toStdString(xml.attributes().value(QStringLiteral("src")).toString());
            xml.skipCurrentElement();
            continue;
        }

        if (xml.isEndElement() && xml.name() == QStringLiteral("channel")) {
            break;
        }
    }

    return channel;
}

XMLTVProgramme parseProgramme(QXmlStreamReader &xml)
{
    XMLTVProgramme programme;
    const QXmlStreamAttributes attributes = xml.attributes();
    programme.channelId = toStdString(attributes.value(QStringLiteral("channel")).toString());
    programme.start = toStdString(attributes.value(QStringLiteral("start")).toString());
    programme.stop = toStdString(attributes.value(QStringLiteral("stop")).toString());

    while (!xml.atEnd()) {
        xml.readNext();

        if (xml.isStartElement() && xml.name() == QStringLiteral("title")) {
            programme.title = toStdString(xml.readElementText().trimmed());
            continue;
        }

        if (xml.isStartElement() && xml.name() == QStringLiteral("desc")) {
            programme.description = toStdString(xml.readElementText().trimmed());
            continue;
        }

        if (xml.isStartElement() && xml.name() == QStringLiteral("icon")) {
            programme.icon = toStdString(xml.attributes().value(QStringLiteral("src")).toString());
            xml.skipCurrentElement();
            continue;
        }

        if (xml.isEndElement() && xml.name() == QStringLiteral("programme")) {
            break;
        }
    }

    return programme;
}

XMLTVGuide parseReader(QXmlStreamReader &xml)
{
    XMLTVGuide guide;
    while (!xml.atEnd()) {
        xml.readNext();

        if (!xml.isStartElement()) {
            continue;
        }

        if (xml.name() == QStringLiteral("channel")) {
            guide.channels.push_back(parseChannel(xml));
            continue;
        }

        if (xml.name() == QStringLiteral("programme")) {
            guide.programmes.push_back(parseProgramme(xml));
            continue;
        }
    }

    if (xml.hasError()) {
        return {};
    }

    return guide;
}

}

XMLTVGuide XMLTVParser::parse(const std::string &filename)
{
    QFile file(QString::fromStdString(filename));
    if (!file.open(QIODevice::ReadOnly | QIODevice::Text)) {
        return {};
    }

    QXmlStreamReader xml(&file);
    return parseReader(xml);
}

XMLTVGuide XMLTVParser::parseContent(const std::string &content)
{
    const QByteArray bytes = QByteArray::fromStdString(content);
    QXmlStreamReader xml(bytes);
    return parseReader(xml);
}
