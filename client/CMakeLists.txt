cmake_minimum_required(VERSION 3.20)

project(retrotv
    VERSION 0.1.0
    DESCRIPTION "Retro cable television client"
    LANGUAGES CXX
)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

find_package(Qt6 REQUIRED COMPONENTS Concurrent Network OpenGLWidgets Widgets)

qt_standard_project_setup()

qt_add_executable(retrotv
    src/main.cpp
    src/app/ChannelTransitionOverlay.h
    src/app/ChannelTransitionOverlay.cpp
    src/app/ClientSettings.h
    src/app/ClientSettings.cpp
    src/app/GuideOverlay.h
    src/app/GuideOverlay.cpp
    src/app/InfoOverlay.h
    src/app/InfoOverlay.cpp
    src/app/MainWindow.h
    src/app/MainWindow.cpp
    src/app/SettingsPage.h
    src/app/SettingsPage.cpp
    src/player/PlayerWidget.h
    src/player/PlayerWidget.cpp
    src/models/Channel.h
    src/models/Channel.cpp
    src/network/ServerDiscovery.h
    src/network/ServerDiscovery.cpp
    src/m3u/M3UParser.h
    src/m3u/M3UParser.cpp
    src/xmltv/XMLTVParser.h
    src/xmltv/XMLTVParser.cpp
)

target_link_libraries(retrotv
    PRIVATE
        Qt6::Network
        Qt6::Concurrent
        Qt6::Widgets
        Qt6::OpenGLWidgets
        mpv
)

target_include_directories(retrotv
    PRIVATE
        src
)
