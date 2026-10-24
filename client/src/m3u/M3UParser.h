#pragma once

#include <string>
#include <vector>

#include "../models/Channel.h"

class M3UParser
{
public:
	static std::vector<Channel>
	parse(
		const std::string& filename
	);

	static std::vector<Channel>
	parseContent(
		const std::string& content
	);
};
