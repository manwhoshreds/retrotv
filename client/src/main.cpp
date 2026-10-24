#include <clocale>

#include <QApplication>
#include <QTimer>

#include "app/MainWindow.h"

int main(int argc, char *argv[])
{
    // On eglfs the pointer is drawn by the platform plugin itself; hide it
    // unless explicitly asked for, since the appliance is driven by a remote.
    if (!qEnvironmentVariableIsSet("QT_QPA_EGLFS_HIDECURSOR")) {
        qputenv("QT_QPA_EGLFS_HIDECURSOR", "1");
    }

    QApplication app(argc, argv);
    QCoreApplication::setOrganizationName(QStringLiteral("RetroTV"));
    QCoreApplication::setApplicationName(QStringLiteral("RetroTV Client"));

    // libmpv requires a locale-independent decimal separator.
    setlocale(LC_NUMERIC, "C");

    MainWindow window;
    if (app.arguments().contains(QStringLiteral("--windowed"))) {
        window.show();
    } else {
        window.showFullScreen();
    }
    QTimer::singleShot(0, &app, []() {
        const QWidgetList topLevelWidgets = QApplication::topLevelWidgets();
        qInfo() << "RetroTV Qt platform" << QGuiApplication::platformName()
                << "top-level widgets" << topLevelWidgets.size();
        if (topLevelWidgets.size() != 1) {
            qWarning() << "RetroTV single-window invariant violated";
        }
    });

    return app.exec();
}
