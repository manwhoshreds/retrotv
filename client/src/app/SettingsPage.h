#pragma once

#include <QWidget>
#include "ClientSettings.h"
#include "network/ServerDiscovery.h"

class QKeyEvent;
class QLabel;
class QLineEdit;
class QPushButton;
class QStackedWidget;
class QVBoxLayout;

class SettingsPage : public QWidget
{
    Q_OBJECT
public:
    explicit SettingsPage(QWidget *parent = nullptr);
    void loadSettings(const ClientSettings &settings, const QList<DiscoveredServer> &servers,
        const QString &activeURL, bool connected);
    void setGuideTextSize(int textSize);
    // Remote/arrow navigation between the section menu and the controls of
    // the open section. Returns true when the key was used.
    bool handleNavigationKey(QKeyEvent *event, bool select);

signals:
    void dismissRequested();
    void saveRequested(const ClientSettings &settings);
    void serverSelected(const QString &url);
    void guideTextSizeChanged(int textSize);

private:
    QPushButton *addSection(const QString &icon, const QString &name);
    void selectSection(int index);
    void showServers(const QList<DiscoveredServer> &servers);
    QList<QWidget *> currentPageControls() const;
    void changeGuideTextSize(int step, bool wrap = false);
    QList<QPushButton *> m_sections;
    QLabel *m_title = nullptr;
    QLabel *m_clock = nullptr;
    QLineEdit *m_serverURL = nullptr;
    QLabel *m_connectionStatus = nullptr;
    QLabel *m_currentServerName = nullptr;
    QLabel *m_currentServerURL = nullptr;
    QWidget *m_manualFallback = nullptr;
    QVBoxLayout *m_serverList = nullptr;
    QStackedWidget *m_pages = nullptr;
    QString m_activeURL;
    QPushButton *m_guideTextSizeControl = nullptr;
    int m_guideTextSize = 0;
};
