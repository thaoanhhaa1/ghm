import SwiftUI

@main
struct GHMApp: App {
    @StateObject private var model = AppModel()
    @StateObject private var updater = UpdaterModel()

    init() {
        UserDefaults.standard.register(defaults: [
            "oauthClientID": "Iv1.b507a08c87ecfe98"
        ])
    }

    var body: some Scene {
        WindowGroup("ghm") {
            RootView()
                .environmentObject(model)
                .environmentObject(updater)
                .frame(minWidth: 980, minHeight: 620)
                .preferredColorScheme(.dark)
        }
        .defaultSize(width: 1220, height: 780)
        .windowStyle(.hiddenTitleBar)
        .windowToolbarStyle(.unified(showsTitle: false))
        .commands {
            CommandGroup(after: .appInfo) {
                Button("Check for Updates…") {
                    updater.checkForUpdates()
                }
                .keyboardShortcut("u", modifiers: [.command, .shift])
            }
            CommandGroup(after: .newItem) {
                Button("Refresh Repositories") {
                    Task { await model.refresh() }
                }
                .keyboardShortcut("r")

                Button("Add Account…") {
                    NotificationCenter.default.post(name: .ghmShowAdd, object: nil)
                }
                .keyboardShortcut("n", modifiers: [.command, .shift])
            }
            CommandGroup(replacing: .help) {
                Button("ghm on GitHub") {
                    NSWorkspace.shared.open(URL(string: "https://github.com/thaoanhhaa1/ghm")!)
                }
            }
        }
    }
}

extension Notification.Name {
    static let ghmShowAdd = Notification.Name("ghm.showAdd")
}
