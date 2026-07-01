// TunaOS Niri Installer — Quickshell + Go installer wizard
//
// Architecture follows DankMaterialShell:
//   - QML (Quickshell) for the UI layer
//   - Go backend that drives fisherman
//
// The installer is a standalone Quickshell window (not a desktop shell).

import QtQuick 6.5
import QtQuick.Controls 6.5
import QtQuick.Layouts 6.5
import QtQuick.Window 6.5

ApplicationWindow {
    id: root
    title: "TunaOS Installer"
    width: 800
    height: 600
    visible: true

    // Backend bridge — calls Go installer backend
    property var backend: InstallerBackend {}

    // Wizard state machine
    property int currentPage: 0  // 0=welcome, 1=disk, 2=confirm, 3=progress, 4=done
    property var disks: []
    property var selectedDisk: ({})
    property string hostname: "tunaos"
    property bool installSuccess: false
    property string installLog: ""

    StackLayout {
        anchors.fill: parent
        currentIndex: currentPage

        // Page 0: Welcome
        ColumnLayout {
            spacing: 20
            anchors.centerIn: parent

            Text {
                text: "TunaOS Installer"
                font.pixelSize: 28
                font.bold: true
                color: "#9ccbfb"
                Layout.alignment: Qt.AlignHCenter
            }
            Text {
                text: "This wizard will guide you through installing TunaOS onto your computer."
                font.pixelSize: 14
                wrapMode: Text.WordWrap
                Layout.maximumWidth: 400
                Layout.alignment: Qt.AlignHCenter
            }
            Button {
                text: "Get Started"
                Layout.alignment: Qt.AlignHCenter
                Layout.preferredWidth: 200
                onClicked: {
                    disks = backend.discoverDisks()
                    currentPage = 1
                }
            }
        }

        // Page 1: Disk Selection
        ColumnLayout {
            spacing: 16
            anchors.fill: parent
            anchors.margins: 40

            Text {
                text: "Select Target Disk"
                font.pixelSize: 22
                font.bold: true
            }
            Text {
                text: "All data on the selected disk will be erased."
                font.pixelSize: 13
                color: "#888"
            }

            ListView {
                Layout.fillWidth: true
                Layout.fillHeight: true
                model: disks
                clip: true
                delegate: ItemDelegate {
                    width: parent.width
                    height: 48
                    text: "/dev/" + modelData.name + "  (" + modelData.size + ")  [" + modelData.transport + "]"
                    highlighted: selectedDisk.name === modelData.name
                    onClicked: selectedDisk = modelData
                }
            }

            RowLayout {
                Layout.fillWidth: true
                Button {
                    text: "Back"
                    onClicked: currentPage = 0
                }
                Item { Layout.fillWidth: true }
                Button {
                    text: "Continue"
                    enabled: selectedDisk.name !== undefined
                    highlighted: true
                    onClicked: currentPage = 2
                }
            }
        }

        // Page 2: Confirm
        ColumnLayout {
            spacing: 12
            anchors.fill: parent
            anchors.margins: 40

            Text {
                text: "Confirm Installation"
                font.pixelSize: 22
                font.bold: true
            }
            GridLayout {
                columns: 2
                columnSpacing: 24
                rowSpacing: 8
                Text { text: "Target Disk:"; font.bold: true }
                Text { text: selectedDisk.name ? "/dev/" + selectedDisk.name : "—" }
                Text { text: "Filesystem:"; font.bold: true }
                Text { text: "xfs" }
                Text { text: "Encryption:"; font.bold: true }
                Text { text: "none" }
                Text { text: "Hostname:"; font.bold: true }
                TextInput { text: hostname; onTextChanged: hostname = text }
                Text { text: "Image:"; font.bold: true }
                Text { text: "ghcr.io/tuna-os/albacore:gnome"; color: "#888"; font.pixelSize: 12 }
            }

            Item { Layout.fillHeight: true }

            RowLayout {
                Layout.fillWidth: true
                Button {
                    text: "Back"
                    onClicked: currentPage = 1
                }
                Item { Layout.fillWidth: true }
                Button {
                    text: "Install"
                    highlighted: true
                    onClicked: {
                        installLog = ""
                        currentPage = 3
                        // Start install in background
                        backend.startInstall(selectedDisk.name, hostname)
                    }
                }
            }
        }

        // Page 3: Install Progress
        ColumnLayout {
            spacing: 12
            anchors.fill: parent
            anchors.margins: 40

            Text {
                text: "Installing..."
                font.pixelSize: 22
                font.bold: true
            }

            ScrollView {
                Layout.fillWidth: true
                Layout.fillHeight: true
                TextArea {
                    text: installLog
                    font.family: "monospace"
                    font.pixelSize: 11
                    readOnly: true
                    wrapMode: TextEdit.Wrap
                }
            }

            Text {
                text: installLog === "" ? "Starting..." : ""
                color: "#888"
                font.italic: true
            }
        }

        // Page 4: Done
        ColumnLayout {
            spacing: 20
            anchors.centerIn: parent

            Text {
                text: installSuccess ? "✓ Installation Complete" : "✗ Installation Failed"
                font.pixelSize: 28
                font.bold: true
                color: installSuccess ? "#33d17a" : "#e66100"
                Layout.alignment: Qt.AlignHCenter
            }
            Text {
                text: installSuccess
                    ? "Remove the installation media and restart your computer."
                    : "Check the installation log for details."
                font.pixelSize: 14
                Layout.alignment: Qt.AlignHCenter
            }
            Button {
                text: "Close"
                Layout.alignment: Qt.AlignHCenter
                Layout.preferredWidth: 200
                onClicked: Qt.quit()
            }
        }
    }

    // Timer to poll install progress from Go backend
    Timer {
        interval: 500
        running: currentPage === 3
        repeat: true
        onTriggered: {
            installLog = backend.pollOutput()
            var status = backend.pollStatus()
            if (status !== "running") {
                installSuccess = (status === "success")
                currentPage = 4
                running = false
            }
        }
    }
}
