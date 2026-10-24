#pragma once

#include <string>
#include <vector>

struct XMLTVChannel
{
    std::string id;
    std::string displayName;
    std::string icon;
};

struct XMLTVProgramme
{
    std::string channelId;
    std::string start;
    std::string stop;
    std::string title;
    std::string description;
    std::string icon;
};

struct XMLTVGuide
{
    std::vector<XMLTVChannel> channels;
    std::vector<XMLTVProgramme> programmes;
};

class XMLTVParser
{
public:
    static XMLTVGuide parse(const std::string &filename);
    static XMLTVGuide parseContent(const std::string &content);
};
