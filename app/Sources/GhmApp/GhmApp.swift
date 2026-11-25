import SwiftUI

@main
struct GHMApp: App {
    @StateObject private var model = AppModel()

    init() {
        UserDefaults.standard.register(defaults: [
            "oauthClientID": "Iv1.b507a08c87ecfe98"
        ])
    }

    var body: some Scene {
        WindowGroup("ghm") {
            RootView()
                .environmentObject(model)
                .frame(minWidth: 920, minHeight: 560)
        }
        .defaultSize(width: 1120, height: 700)
        .windowToolbarStyle(.unified)
        .commands {
            CommandGroup(after: .newItem) {
                Button("Refresh Repositories") {
                    Task { await model.refresh() }
                }
                .keyboardShortcut("r")
            }
            CommandGroup(replacing: .help) {
                Button("ghm on GitHub") {
                    NSWorkspace.shared.open(URL(string: "https://github.com/thaoanhhaa1/ghm")!)
                }
            }
        }
    }
}
