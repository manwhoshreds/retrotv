#pragma once

#include <QWidget>
#include <QDateTime>
#include <QHash>
#include <vector>

#include "xmltv/XMLTVParser.h"

class QKeyEvent;
class QFrame;
class QGridLayout;
class QLabel;
class QScrollArea;

class GuideOverlay : public QWidget
{
    Q_OBJECT

public:
    // Guide text size: the programme cell font size in px. All other guide
    // text and layout scales with it.
    static constexpr int MinimumTextSize = 16;
    static constexpr int MaximumTextSize = 40;
    static constexpr int DefaultTextSize = 28;

    explicit GuideOverlay(QWidget *parent = nullptr);
    void setGuide(XMLTVGuide guide);
    void setCurrentChannelId(int channelId);
    void resetTimelineToNow();
    void scrollChannels(int offset);
    void pageChannels(int direction);
    void scrollTimeline(int offset);
    int selectedChannelId() const;
    void setTextSize(int textSize);
    int textSize() const;

signals:
    void dismissRequested();
    void channelSelected(int channelId);

protected:
    void keyPressEvent(QKeyEvent *event) override;

private:
    struct PreparedProgramme {
        const XMLTVProgramme *programme = nullptr;
        QDateTime start;
        QDateTime stop;
    };

    void prepareGuide();
    const std::vector<PreparedProgramme> *programmesForChannel(const std::string &channelId) const;
    const PreparedProgramme *programmeAtTime(int channelId, const QDateTime &selectedTime) const;
    const PreparedProgramme *nextProgrammeForChannel(const PreparedProgramme *current) const;
    QDateTime clampedTimelineStart(QDateTime value) const;
    void rebuildUi();
    int selectedChannelIndex() const;
    int visibleRowCapacity() const;
    void keepSelectionVisible(int visibleRows);
    void scrollToSelectedChannel();
    void updateSelectionStyles();
    void updateSelectedProgrammeDetails();
    void updateTimelineWindow();
    QLabel *acquireProgramCell(const XMLTVProgramme *programme, const QString &objectName);
    void recycleTimelineCells();

    XMLTVGuide m_guide;
    QHash<QString, std::vector<PreparedProgramme>> m_programmesByChannel;
    QDateTime m_guideStop;
    QDateTime m_timelineStart;
    int m_textSize = DefaultTextSize;
    int m_currentChannelId = 0;
    int m_selectedChannelId = 0;
    int m_channelOffset = 0;
    QFrame *m_gridFrame = nullptr;
    QGridLayout *m_gridLayout = nullptr;
    QScrollArea *m_gridScrollArea = nullptr;
    std::vector<QLabel *> m_timeHeaderLabels;
    std::vector<QLabel *> m_reusableProgramCells;
    QLabel *m_detailPosterLabel = nullptr;
    QLabel *m_detailTitleLabel = nullptr;
    QLabel *m_detailTimeLabel = nullptr;
    QLabel *m_detailDescriptionLabel = nullptr;
    QLabel *m_upNextTitleLabel = nullptr;
    QLabel *m_upNextTimeLabel = nullptr;
};
