#include "SettingsPage.h"

#include <QApplication>
#include <QDateTime>
#include <QFrame>
#include <QFutureWatcher>
#include <QHBoxLayout>
#include <QKeyEvent>
#include <QLabel>
#include <QLineEdit>
#include <QPushButton>
#include <QStackedWidget>
#include <QStyle>
#include <QTimer>
#include <QVBoxLayout>
#include <QtConcurrent>

#include "GuideOverlay.h"

namespace {
QFrame *rule(QWidget *parent, const char *name)
{
    auto *line = new QFrame(parent);
    line->setObjectName(QString::fromLatin1(name));
    line->setFrameShape(QFrame::HLine);
    return line;
}

QWidget *placeholder(const QString &text, QWidget *parent)
{
    auto *page = new QWidget(parent);
    auto *layout = new QVBoxLayout(page);
    layout->addStretch();
    auto *label = new QLabel(text, page);
    label->setObjectName(QStringLiteral("placeholder"));
    label->setAlignment(Qt::AlignCenter);
    layout->addWidget(label);
    layout->addStretch();
    return page;
}
}

SettingsPage::SettingsPage(QWidget *parent) : QWidget(parent)
{
    setObjectName(QStringLiteral("settingsPage"));
    auto *root = new QVBoxLayout(this);
    root->setContentsMargins(46, 34, 46, 26);
    root->setSpacing(22);

    auto *header = new QHBoxLayout();
    auto *brand = new QLabel(QStringLiteral("▣ RETROTV"), this);
    brand->setObjectName(QStringLiteral("brand"));
    auto *heading = new QLabel(QStringLiteral("SETTINGS"), this);
    heading->setObjectName(QStringLiteral("heading"));
    heading->setAlignment(Qt::AlignCenter);
    m_clock = new QLabel(this);
    m_clock->setObjectName(QStringLiteral("clock"));
    m_clock->setAlignment(Qt::AlignRight | Qt::AlignVCenter);
    header->addWidget(brand, 1);
    header->addWidget(heading, 1);
    header->addWidget(m_clock, 1);
    root->addLayout(header);
    root->addWidget(rule(this, "headerRule"));

    auto *body = new QHBoxLayout();
    body->setSpacing(20);
    auto *menu = new QWidget(this);
    menu->setObjectName(QStringLiteral("settingsMenu"));
    menu->setFixedWidth(420);
    auto *menuLayout = new QVBoxLayout(menu);
    menuLayout->setContentsMargins(0, 0, 0, 0);
    menuLayout->setSpacing(8);
    const QList<QPair<QString, QString>> sections = {
        {QStringLiteral("♒"), QStringLiteral("CONNECTION")},
        {QStringLiteral("▣"), QStringLiteral("PICTURE")},
        {QStringLiteral("◖"), QStringLiteral("AUDIO")},
        {QStringLiteral("▦"), QStringLiteral("GUIDE")},
        {QStringLiteral("◆"), QStringLiteral("REMOTE & DEVICES")},
        {QStringLiteral("▣"), QStringLiteral("PARENTAL CONTROLS")},
        {QStringLiteral("●"), QStringLiteral("ABOUT")},
    };
    for (const auto &section : sections) menuLayout->addWidget(addSection(section.first, section.second));
    menuLayout->addStretch();
    body->addWidget(menu);

    auto *panel = new QFrame(this);
    panel->setObjectName(QStringLiteral("panel"));
    auto *panelLayout = new QVBoxLayout(panel);
    panelLayout->setContentsMargins(30, 24, 30, 24);
    m_title = new QLabel(QStringLiteral("CONNECTION"), panel);
    m_title->setObjectName(QStringLiteral("panelTitle"));
    panelLayout->addWidget(m_title);
    panelLayout->addWidget(rule(panel, "panelRule"));
    auto *description = new QLabel(QStringLiteral("Connect RetroTV to a server on your network."), panel);
    description->setObjectName(QStringLiteral("description"));
    panelLayout->addWidget(description);
    m_pages = new QStackedWidget(panel);

    auto *server = new QWidget(panel);
    auto *serverLayout = new QVBoxLayout(server);
    serverLayout->setContentsMargins(4, 22, 4, 4);
    serverLayout->setSpacing(20);
    auto *connectionLabel = new QLabel(QStringLiteral("CURRENT CONNECTION"), server);
    connectionLabel->setObjectName(QStringLiteral("rowLabel"));
    serverLayout->addWidget(connectionLabel);
    auto *currentCard = new QFrame(server);
    currentCard->setObjectName(QStringLiteral("currentCard"));
    auto *connection = new QHBoxLayout(currentCard);
    auto *currentDot = new QLabel(QStringLiteral("●"), currentCard);
    currentDot->setObjectName(QStringLiteral("currentDot"));
    auto *currentCopy = new QVBoxLayout();
    m_currentServerName = new QLabel(QStringLiteral("Not Connected"), currentCard);
    m_currentServerName->setObjectName(QStringLiteral("currentName"));
    m_currentServerURL = new QLabel(QStringLiteral("No server selected"), currentCard);
    m_currentServerURL->setObjectName(QStringLiteral("currentURL"));
    currentCopy->addWidget(m_currentServerName);
    currentCopy->addWidget(m_currentServerURL);
    m_connectionStatus = new QLabel(QStringLiteral("SEARCHING..."), server);
    m_connectionStatus->setObjectName(QStringLiteral("configured"));
    connection->addWidget(currentDot);
    connection->addLayout(currentCopy);
    connection->addStretch();
    connection->addWidget(m_connectionStatus);
    serverLayout->addWidget(currentCard);
    auto *detectedLabel = new QLabel(QStringLiteral("AVAILABLE SERVERS"), server);
    detectedLabel->setObjectName(QStringLiteral("rowLabel"));
    serverLayout->addWidget(detectedLabel);
    auto *serverListWidget = new QWidget(server);
    m_serverList = new QVBoxLayout(serverListWidget);
    m_serverList->setContentsMargins(0, 0, 0, 0);
    m_serverList->setSpacing(6);
    serverLayout->addWidget(serverListWidget);
    auto *rescan = new QPushButton(QStringLiteral("SCAN FOR SERVERS     ↻"), server);
    rescan->setObjectName(QStringLiteral("rescan"));
    connect(rescan, &QPushButton::clicked, this, [this, rescan]() {
        rescan->setEnabled(false);
        rescan->setText(QStringLiteral("SCANNING..."));
        auto *watcher = new QFutureWatcher<QList<DiscoveredServer>>(this);
        connect(watcher, &QFutureWatcher<QList<DiscoveredServer>>::finished, this, [this, watcher, rescan]() {
            showServers(watcher->result());
            watcher->deleteLater();
            rescan->setEnabled(true);
            rescan->setText(QStringLiteral("SCAN FOR SERVERS     ↻"));
        });
        watcher->setFuture(QtConcurrent::run([]() { return discoverRetroTVServers(); }));
    });
    serverLayout->addWidget(rescan);
    m_manualFallback = new QWidget(server);
    auto *manualLayout = new QVBoxLayout(m_manualFallback);
    manualLayout->setContentsMargins(0, 0, 0, 0);
    auto *address = new QLabel(QStringLiteral("MANUAL FALLBACK ADDRESS"), server);
    address->setObjectName(QStringLiteral("rowLabel"));
    manualLayout->addWidget(address);
    m_serverURL = new QLineEdit(server);
    m_serverURL->setObjectName(QStringLiteral("serverURL"));
    m_serverURL->setPlaceholderText(QStringLiteral("http://192.168.1.50:8080"));
    manualLayout->addWidget(m_serverURL);
    auto *save = new QPushButton(QStringLiteral("SAVE FALLBACK & RETRY"), server);
    save->setObjectName(QStringLiteral("save"));
    connect(save, &QPushButton::clicked, this, [this]() {
        QString url = m_serverURL->text().trimmed();
        while (url.endsWith('/')) url.chop(1);
        if (!url.startsWith("http://", Qt::CaseInsensitive) && !url.startsWith("https://", Qt::CaseInsensitive)) {
            m_serverURL->setFocus(); m_serverURL->selectAll(); return;
        }
        ClientSettings value; value.backendURL = url; emit saveRequested(value);
    });
    manualLayout->addWidget(save, 0, Qt::AlignLeft);
    serverLayout->addWidget(m_manualFallback);
    serverLayout->addStretch();
    m_pages->addWidget(server);
    m_pages->addWidget(placeholder(QStringLiteral("PICTURE CONTROLS\nCOMING SOON"), panel));
    m_pages->addWidget(placeholder(QStringLiteral("AUDIO CONTROLS\nCOMING SOON"), panel));
    auto *guide = new QWidget(panel);
    auto *guideLayout = new QVBoxLayout(guide);
    guideLayout->setContentsMargins(4, 22, 4, 4);
    guideLayout->setSpacing(14);
    auto *textSizeLabel = new QLabel(QStringLiteral("TEXT SIZE"), guide);
    textSizeLabel->setObjectName(QStringLiteral("rowLabel"));
    guideLayout->addWidget(textSizeLabel);
    m_guideTextSizeControl = new QPushButton(guide);
    m_guideTextSizeControl->setObjectName(QStringLiteral("stepper"));
    connect(m_guideTextSizeControl, &QPushButton::clicked, this, [this]() { changeGuideTextSize(1, true); });
    guideLayout->addWidget(m_guideTextSizeControl, 0, Qt::AlignLeft);
    auto *textSizeHint = new QLabel(QStringLiteral("◀ ▶ to change. The guide updates right away."), guide);
    textSizeHint->setObjectName(QStringLiteral("description"));
    guideLayout->addWidget(textSizeHint);
    guideLayout->addStretch();
    m_pages->addWidget(guide);
    m_pages->addWidget(placeholder(QStringLiteral("REMOTE PAIRING\nCOMING SOON"), panel));
    m_pages->addWidget(placeholder(QStringLiteral("PARENTAL CONTROLS\nCOMING SOON"), panel));
    m_pages->addWidget(placeholder(QStringLiteral("RETROTV CLIENT  v0.1.0"), panel));
    panelLayout->addWidget(m_pages, 1);
    body->addWidget(panel, 1);
    root->addLayout(body, 1);

    auto *footer = new QHBoxLayout();
    footer->addWidget(new QLabel(QStringLiteral("▲▼  NAVIGATE"), this));
    footer->addStretch();
    footer->addWidget(new QLabel(QStringLiteral("[ OK ]  SELECT"), this));
    footer->addStretch();
    auto *back = new QPushButton(QStringLiteral("↩  BACK"), this);
    back->setObjectName(QStringLiteral("back"));
    connect(back, &QPushButton::clicked, this, &SettingsPage::dismissRequested);
    footer->addWidget(back);
    root->addLayout(footer);

    auto *timer = new QTimer(this);
    connect(timer, &QTimer::timeout, this, [this]() { m_clock->setText(QTime::currentTime().toString("hh:mm AP")); });
    timer->start(1000);
    m_clock->setText(QTime::currentTime().toString("hh:mm AP"));

    setStyleSheet(QStringLiteral(R"(
      QWidget#settingsPage { background: qradialgradient(cx:.5,cy:.15,radius:1,stop:0 #071024,stop:.7 #020510,stop:1 #090018); color:#f5f7ff; }
      QLabel#brand { color:#f044bd; font-size:32px; font-weight:900; font-style:italic; }
      QLabel#heading { color:#ffe829; font-size:38px; font-weight:900; letter-spacing:4px; }
      QLabel#clock { color:#52a8ff; font-size:23px; font-weight:900; }
      QFrame#headerRule { border:0; min-height:3px; max-height:3px; background:#c42eff; }
      QWidget#settingsMenu { border:2px solid #155dcc; border-radius:8px; background:#020611; }
      QPushButton[nav="true"] { min-height:68px; padding:0 18px; border:2px solid #103d85; border-radius:8px; background:#030815; color:#f5f7ff; text-align:left; font-size:21px; font-weight:900; }
      QPushButton[nav="true"]:focus, QPushButton[nav="true"]:hover, QPushButton[nav="true"][selected="true"] { border-color:#ff43bc; background:#2d092c; }
      QFrame#panel { border:2px solid #155dcc; border-radius:10px; background:#020819; }
      QLabel#panelTitle { color:#ff43bc; font-size:31px; font-weight:900; }
      QLabel#description { color:#c8d2e3; font-size:17px; padding:4px 0 8px 0; }
      QFrame#panelRule, QFrame#rowRule { border:0; min-height:2px; max-height:2px; background:#155dcc; }
      QLabel#rowLabel { font-size:20px; font-weight:900; } QLabel#configured { color:#42e35a; font-size:20px; font-weight:900; }
      QFrame#currentCard { min-height:66px; border:2px solid #3a8f19; border-radius:7px; background:#061108; }
      QLabel#currentDot { color:#42e35a; font-size:24px; } QLabel#currentName { color:#f5f7ff; font-size:20px; font-weight:900; }
      QLabel#currentURL { color:#8fa0b8; font-size:16px; }
      QPushButton[server="true"] { min-height:54px; padding:6px 14px; border:1px solid #29405d; border-radius:5px; background:#050811; color:#f5f7ff; text-align:left; font-size:18px; font-weight:800; }
      QPushButton[server="true"]:hover, QPushButton[server="true"]:focus { border-color:#29d7ff; background:#07172a; }
      QPushButton#rescan { min-height:44px; border:1px solid #29405d; background:#050811; color:#ffe829; text-align:left; padding-left:14px; font-size:17px; font-weight:900; }
      QLabel#placeholder { color:#56a9ff; font-size:26px; font-weight:900; }
      QLineEdit#serverURL { min-height:50px; padding:0 14px; border:2px solid #155dcc; border-radius:6px; background:#01050d; color:#56a9ff; font-size:20px; }
      QLineEdit#serverURL:focus { border-color:#ff43bc; }
      QPushButton#save { min-height:50px; padding:0 24px; border:2px solid #ff43bc; border-radius:6px; background:#410b3b; color:white; font-size:19px; font-weight:900; }
      QPushButton#stepper { min-height:58px; min-width:320px; padding:0 22px; border:2px solid #155dcc; border-radius:6px; background:#01050d; color:#ffe829; font-size:24px; font-weight:900; }
      QPushButton#stepper:focus, QPushButton#stepper:hover { border-color:#ff43bc; background:#2d092c; }
      QPushButton#back { border:0; background:transparent; color:#55aaff; font-size:18px; font-weight:900; }
    )"));
    selectSection(0);
}

QPushButton *SettingsPage::addSection(const QString &icon, const QString &name)
{
    int index = m_sections.size();
    auto *button = new QPushButton(QStringLiteral("  %1     %2                         ›").arg(icon, name), this);
    button->setProperty("nav", true);
    connect(button, &QPushButton::clicked, this, [this, index]() { selectSection(index); });
    m_sections.append(button);
    return button;
}

void SettingsPage::selectSection(int index)
{
    static const QStringList names = {"CONNECTION", "PICTURE", "AUDIO", "GUIDE", "REMOTE & DEVICES", "PARENTAL CONTROLS", "ABOUT"};
    for (int i = 0; i < m_sections.size(); ++i) {
        m_sections[i]->setProperty("selected", i == index);
        m_sections[i]->style()->unpolish(m_sections[i]); m_sections[i]->style()->polish(m_sections[i]);
    }
    m_title->setText(names[index]); m_pages->setCurrentIndex(index); m_sections[index]->setFocus();
}

void SettingsPage::loadSettings(const ClientSettings &settings, const QList<DiscoveredServer> &servers,
    const QString &activeURL, bool connected)
{
    m_serverURL->setText(settings.backendURL);
    m_activeURL = activeURL;
    showServers(servers);
    if (connected) {
        bool usingDiscovered = false;
        QString activeName = QStringLiteral("Manual RetroTV Server");
        for (const DiscoveredServer &server : servers) {
            if (server.url == activeURL) {
                usingDiscovered = true;
                activeName = server.name;
            }
        }
        m_currentServerName->setText(activeName);
        m_currentServerURL->setText(activeURL);
        m_connectionStatus->setText(usingDiscovered
            ? QStringLiteral("AUTO CONNECTED  ●")
            : QStringLiteral("MANUAL CONNECTED  ●"));
        m_connectionStatus->setStyleSheet(QStringLiteral("color:#42e35a"));
    } else {
        m_currentServerName->setText(QStringLiteral("Not Connected"));
        m_currentServerURL->setText(QStringLiteral("No server selected"));
        m_connectionStatus->setText(QStringLiteral("DISCONNECTED  ●"));
        m_connectionStatus->setStyleSheet(QStringLiteral("color:#ff4a54"));
    }
    m_manualFallback->setVisible(!connected && servers.isEmpty());
    selectSection(0);
}

void SettingsPage::showServers(const QList<DiscoveredServer> &servers)
{
    while (QLayoutItem *item = m_serverList->takeAt(0)) {
        delete item->widget();
        delete item;
    }
    int availableCount = 0;
    if (!servers.isEmpty()) {
        for (const DiscoveredServer &server : servers) {
            if (server.url == m_activeURL) continue;
            ++availableCount;
            auto *button = new QPushButton(QStringLiteral("%1\n%2                                      ›").arg(server.name, server.url), this);
            button->setProperty("server", true);
            connect(button, &QPushButton::clicked, this, [this, server]() { emit serverSelected(server.url); });
            m_serverList->addWidget(button);
        }
    }
    if (availableCount == 0) {
        auto *empty = new QLabel(m_activeURL.isEmpty()
            ? QStringLiteral("No RetroTV servers detected")
            : QStringLiteral("No other RetroTV servers detected"), this);
        empty->setObjectName(QStringLiteral("placeholder"));
        m_serverList->addWidget(empty);
    }
}

void SettingsPage::setGuideTextSize(int textSize)
{
    m_guideTextSize = textSize;
    m_guideTextSizeControl->setText(QStringLiteral("◀      %1 px      ▶").arg(textSize));
}

void SettingsPage::changeGuideTextSize(int step, bool wrap)
{
    constexpr int kStepPixels = 2;
    int next = m_guideTextSize + step * kStepPixels;
    // OK steps upwards and wraps around, so the remote can cycle sizes.
    if (wrap && step > 0 && m_guideTextSize >= GuideOverlay::MaximumTextSize) {
        next = GuideOverlay::MinimumTextSize;
    }
    next = std::clamp(next, GuideOverlay::MinimumTextSize, GuideOverlay::MaximumTextSize);
    if (next == m_guideTextSize) {
        return;
    }
    setGuideTextSize(next);
    emit guideTextSizeChanged(next);
}

QList<QWidget *> SettingsPage::currentPageControls() const
{
    QList<QWidget *> controls;
    if (!m_pages->currentWidget()) {
        return controls;
    }
    for (QWidget *widget : m_pages->currentWidget()->findChildren<QWidget *>()) {
        if (widget->isVisible() && widget->isEnabled() && (widget->focusPolicy() & Qt::TabFocus)) {
            controls.append(widget);
        }
    }
    return controls;
}

bool SettingsPage::handleNavigationKey(QKeyEvent *event, bool select)
{
    QWidget *focus = QApplication::focusWidget();
    const int key = event->key();

    const int section = m_sections.indexOf(qobject_cast<QPushButton *>(focus));
    if (section >= 0) {
        if (key == Qt::Key_Up) {
            selectSection(std::max(0, section - 1));
        } else if (key == Qt::Key_Down) {
            selectSection(std::min(static_cast<int>(m_sections.size()) - 1, section + 1));
        } else if (key == Qt::Key_Right || select) {
            const QList<QWidget *> controls = currentPageControls();
            if (!controls.isEmpty()) {
                controls.front()->setFocus(Qt::TabFocusReason);
            }
        } else {
            return key == Qt::Key_Left;
        }
        return true;
    }

    const QList<QWidget *> controls = currentPageControls();
    const int index = controls.indexOf(focus);
    if (index < 0) {
        return false;
    }
    const bool textField = qobject_cast<QLineEdit *>(focus) != nullptr;

    if (key == Qt::Key_Up) {
        if (index == 0) {
            m_sections[m_pages->currentIndex()]->setFocus(Qt::TabFocusReason);
        } else {
            controls.at(index - 1)->setFocus(Qt::TabFocusReason);
        }
        return true;
    }
    if (key == Qt::Key_Down) {
        if (index + 1 < controls.size()) {
            controls.at(index + 1)->setFocus(Qt::TabFocusReason);
        }
        return true;
    }
    if (focus == m_guideTextSizeControl && (key == Qt::Key_Left || key == Qt::Key_Right)) {
        changeGuideTextSize(key == Qt::Key_Right ? 1 : -1);
        return true;
    }
    if (key == Qt::Key_Left && !textField) {
        m_sections[m_pages->currentIndex()]->setFocus(Qt::TabFocusReason);
        return true;
    }
    if (key == Qt::Key_Right && !textField) {
        return true;
    }
    if (select && !textField) {
        if (auto *button = qobject_cast<QAbstractButton *>(focus)) {
            button->animateClick();
        }
        return true;
    }
    return false;
}
