#include <fstream>
#include <istream>
#include <sstream>

#include "M3UParser.h"

namespace {

std::vector<Channel>
parseStream(std::istream &stream)
{
    std::vector<Channel> channels;

    std::string line;
    int channelNumber = 1;

    while (std::getline(stream, line))
    {
       if (line.starts_with("#EXTINF")) 
       {
        std::size_t commaPos = line.find(",");
        std::string channelName = line.substr(commaPos + 1);

        std::string mediaPath;
        std::getline(stream, mediaPath);

        channels.push_back(
            {
                0,
                channelNumber,
                channelName,
                mediaPath
            }
        );

        channelNumber++;

       }
    }
    
    return channels;
}

}

std::vector<Channel>
M3UParser::parse(const std::string& filename)
{
    std::ifstream file(filename);
    if (!file)
    {
        return {};
    }

    return parseStream(file);
}

std::vector<Channel>
M3UParser::parseContent(const std::string& content)
{
    std::istringstream stream(content);
    return parseStream(stream);
}
