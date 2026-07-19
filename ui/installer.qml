// TunaOS Niri Installer — Quickshell + Go installer wizard
//
// QML (Quickshell) UI layer over the Go backend in ../installer, which wraps
// fisherman. Backend binary is resolved from $TUNA_BACKEND or PATH
// ("tuna-installer-backend"; the Flatpak installs it at /app/bin).
//
// Visual design target: ../DESIGN.md (scrolling column strip, instrument
// panel). This file is the functional wizard; the strip treatment lands on top.

import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
import Quickshell
import Quickshell.Io

ApplicationWindow {
    id: root
    title: "TunaOS Installer"
    width: 800
    height: 600
    visible: true
    color: "#0A0E12" // --void (DESIGN.md)

    property string backendBin: Quickshell.env("TUNA_BACKEND") || "tuna-installer-backend"

    // Wizard state
    property int currentPage: 0 // 0=welcome, 1=disk, 2=encryption, 3=confirm, 4=progress, 5=done
    // Encryption was previously hardcoded to "none" in the recipe with no UI,
    // so every install came out unencrypted (tuna-os/tunaOS#734).
    property string encType: "none"
    property string passphrase: ""
    property bool hasTpm: false
    property var disks: []
    property var selectedDisk: ({})
    property string hostname: "tunaos"
    property bool installSuccess: false
    property string installLog: ""

    // Offline facts from `detect` (spec §4)
    property string liveImage: ""
    property var offlineStores: []
    property string defaultImage: "ghcr.io/tuna-os/albacore:gnome"

    Component.onCompleted: detectProc.running = true

    Process {
        id: detectProc
        command: [root.backendBin, "detect"]
        stdout: StdioCollector {
            onStreamFinished: {
                try {
                    const facts = JSON.parse(text)
                    root.liveImage = facts.liveImage || ""
                    root.hasTpm = facts.hasTpm === true
                    root.offlineStores = facts.offlineStores || []
                } catch (e) { /* detect is best-effort */ }
            }
        }
    }

    Process {
        id: discoverProc
        command: [root.backendBin, "discover-disks"]
        stdout: StdioCollector {
            onStreamFinished: {
                try { root.disks = JSON.parse(text) } catch (e) { root.disks = [] }
            }
        }
    }

    Process {
        id: installProc
        stdout: SplitParser {
            onRead: data => root.installLog += data + "\n"
        }
        stderr: SplitParser {
            onRead: data => root.installLog += data + "\n"
        }
        onExited: (code, status) => {
            root.installSuccess = (code === 0)
            root.currentPage = 5
        }
    }

    function startInstall() {
        installLog = ""
        currentPage = 4
        const recipe = {
            disk: "/dev/" + selectedDisk.name,
            filesystem: "xfs",
            encryption: root.encType.endsWith("passphrase")
                ? { type: root.encType, passphrase: root.passphrase }
                : { type: root.encType },
            // Empty image = live-ISO self-install (bootc uses the running container)
            image: liveImage !== "" ? "" : defaultImage,
            hostname: hostname,
            distroID: "tunaos",
            selinuxDisabled: true,
            additionalImageStores: offlineStores
        }
        installProc.command = [root.backendBin, "install", JSON.stringify(recipe)]
        installProc.running = true
    }

    StackLayout {
        anchors.fill: parent
        currentIndex: currentPage

        // Page 0: Welcome
        Item {
            ColumnLayout {
                spacing: 20
                anchors.centerIn: parent

                Text {
                    text: "TunaOS Installer"
                    font.pixelSize: 28
                    font.weight: Font.Light
                    color: "#2EC4B6" // --sonar
                    Layout.alignment: Qt.AlignHCenter
                }
                Text {
                    text: root.liveImage !== ""
                        ? "Install this system — no download required."
                        : "This wizard will guide you through installing TunaOS onto your computer."
                    font.pixelSize: 14
                    color: "#8FA3B0" // --fog
                    wrapMode: Text.WordWrap
                    Layout.maximumWidth: 420
                    Layout.alignment: Qt.AlignHCenter
                }
                Button {
                    text: "Get Started"
                    Layout.alignment: Qt.AlignHCenter
                    Layout.preferredWidth: 200
                    onClicked: {
                        discoverProc.running = true
                        root.currentPage = 1
                    }
                }
            }
        }

        // Page 1: Disk Selection
        ColumnLayout {
            spacing: 16
            anchors.margins: 40

            Text {
                text: "Destination"
                font.pixelSize: 22
                font.weight: Font.Light
                color: "#8FA3B0"
            }
            Text {
                text: root.selectedDisk.name !== undefined
                    ? "erases everything on " + root.selectedDisk.name
                    : "All data on the selected disk will be erased."
                font.pixelSize: 13
                color: "#F4A259" // --catch
            }

            ListView {
                Layout.fillWidth: true
                Layout.fillHeight: true
                model: root.disks
                clip: true
                delegate: ItemDelegate {
                    width: ListView.view.width
                    height: 48
                    text: "/dev/" + modelData.name + "  (" + modelData.size + ")  [" + (modelData.tran || "?") + "]"
                    font.family: "monospace"
                    highlighted: root.selectedDisk.name === modelData.name
                    onClicked: root.selectedDisk = modelData
                }
            }

            RowLayout {
                Layout.fillWidth: true
                Button { text: "Back"; onClicked: root.currentPage = 0 }
                Item { Layout.fillWidth: true }
                Button {
                    text: "Continue"
                    enabled: root.selectedDisk.name !== undefined
                    highlighted: true
                    onClicked: root.currentPage = 2
                }
            }
        }

        // Page 2: Encryption
        //
        // Options mirror tuna-installer-xfce's ENCRYPTION_CHOICES, which is the
        // reference implementation — same values, same wording, so the
        // frontends describe the same choice identically. The tpm2 options are
        // omitted entirely when the backend reports no TPM, rather than shown
        // and then failing at install time.
        ColumnLayout {
            spacing: 12
            anchors.margins: 40

            Text {
                text: "Disk Encryption"
                font.pixelSize: 28; font.bold: true; color: "white"
            }
            Text {
                text: "Encryption protects your files if the disk is lost or stolen. It cannot be turned on later without reinstalling."
                wrapMode: Text.WordWrap
                Layout.fillWidth: true
                color: "#8FA3B0"
            }

            ButtonGroup { id: encGroup }

            Repeater {
                model: [
                    { value: "none",                 label: "No encryption",    explain: "Anyone with the disk can read your files." },
                    { value: "luks-passphrase",      label: "Passphrase",       explain: "You'll type it at every boot." },
                    { value: "tpm2-luks",            label: "TPM",              explain: "Unlocks automatically on this hardware." },
                    { value: "tpm2-luks-passphrase", label: "TPM + passphrase", explain: "Automatic unlock, passphrase as fallback." }
                ]
                ColumnLayout {
                    spacing: 2
                    visible: !modelData.value.startsWith("tpm2") || root.hasTpm
                    RadioButton {
                        text: modelData.label
                        ButtonGroup.group: encGroup
                        checked: root.encType === modelData.value
                        onClicked: root.encType = modelData.value
                        palette.windowText: "white"
                    }
                    Text {
                        text: modelData.explain
                        color: "#5A6B78"; leftPadding: 32
                    }
                }
            }

            // Only meaningful for the *-passphrase modes.
            ColumnLayout {
                visible: root.encType.endsWith("passphrase")
                spacing: 6
                Layout.leftMargin: 32
                TextField {
                    id: passField
                    placeholderText: "Enter passphrase"
                    echoMode: TextInput.Password
                    Layout.preferredWidth: 320
                    onTextChanged: root.passphrase = text
                }
                TextField {
                    id: passConfirm
                    placeholderText: "Confirm passphrase"
                    echoMode: TextInput.Password
                    Layout.preferredWidth: 320
                }
                Text {
                    id: passError
                    color: "#C0392B"
                    visible: text !== ""
                }
            }

            Item { Layout.fillHeight: true }

            RowLayout {
                Button { text: "Back"; onClicked: root.currentPage = 1 }
                Item { Layout.fillWidth: true }
                Button {
                    text: "Continue"
                    onClicked: {
                        // fisherman rejects a *-passphrase type with an empty
                        // passphrase, but that only surfaces mid-install; catch
                        // it here where it can still be corrected.
                        if (root.encType.endsWith("passphrase")) {
                            if (passField.text === "") {
                                passError.text = "Enter a passphrase."
                                return
                            }
                            if (passField.text !== passConfirm.text) {
                                passError.text = "Passphrases do not match."
                                return
                            }
                        }
                        passError.text = ""
                        root.passphrase = passField.text
                        root.currentPage = 3
                    }
                }
            }
        }

        // Page 3: Confirm
        ColumnLayout {
            spacing: 12
            anchors.margins: 40

            Text {
                text: "Confirm Installation"
                font.pixelSize: 22
                font.weight: Font.Light
                color: "#8FA3B0"
            }
            GridLayout {
                columns: 2
                columnSpacing: 24
                rowSpacing: 8
                Text { text: "Target Disk:"; font.bold: true; color: "#8FA3B0" }
                Text {
                    text: root.selectedDisk.name ? "/dev/" + root.selectedDisk.name : "—"
                    font.family: "monospace"; color: "white"
                }
                Text { text: "Filesystem:"; font.bold: true; color: "#8FA3B0" }
                Text { text: "xfs"; font.family: "monospace"; color: "white" }
                Text { text: "Encryption:"; font.bold: true; color: "#8FA3B0" }
                Text { text: root.encType; font.family: "monospace"; color: "white" }
                Text { text: "Hostname:"; font.bold: true; color: "#8FA3B0" }
                TextField {
                    text: root.hostname
                    onTextChanged: root.hostname = text
                    font.family: "monospace"
                }
                Text { text: "Image:"; font.bold: true; color: "#8FA3B0" }
                Text {
                    text: root.liveImage !== ""
                        ? root.liveImage + "  (this system, no download)"
                        : root.defaultImage
                    color: "#8FA3B0"; font.pixelSize: 12; font.family: "monospace"
                }
            }

            Item { Layout.fillHeight: true }

            RowLayout {
                Layout.fillWidth: true
                Button { text: "Back"; onClicked: root.currentPage = 2 }
                Item { Layout.fillWidth: true }
                Button {
                    text: "Install"
                    highlighted: true
                    onClicked: root.startInstall()
                }
            }
        }

        // Page 4: Install Progress
        ColumnLayout {
            spacing: 12
            anchors.margins: 40

            Text {
                text: "Installing…"
                font.pixelSize: 22
                font.weight: Font.Light
                color: "#8FA3B0"
            }

            ScrollView {
                Layout.fillWidth: true
                Layout.fillHeight: true
                TextArea {
                    text: root.installLog === "" ? "Starting…" : root.installLog
                    font.family: "monospace"
                    font.pixelSize: 11
                    readOnly: true
                    wrapMode: TextEdit.Wrap
                }
            }
        }

        // Page 5: Done
        Item {
            ColumnLayout {
                spacing: 20
                anchors.centerIn: parent

                Text {
                    text: root.installSuccess ? "✓ Installation Complete" : "✗ Installation Failed"
                    font.pixelSize: 28
                    font.weight: Font.Light
                    color: root.installSuccess ? "#2EC4B6" : "#F4A259"
                    Layout.alignment: Qt.AlignHCenter
                }
                Text {
                    text: root.installSuccess
                        ? "Remove the installation media and restart your computer."
                        : "Check the installation log above for details."
                    font.pixelSize: 14
                    color: "#8FA3B0"
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
    }
}
