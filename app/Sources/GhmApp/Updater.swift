import Foundation
import Sparkle

/// Feed overrides for staging/tests and for enterprise builds. Sparkle asks the
/// delegate first and falls back to `SUFeedURL` in Info.plist.
private final class SparkleDelegate: NSObject, SPUUpdaterDelegate {
    func feedURLString(for updater: SPUUpdater) -> String? {
        ProcessInfo.processInfo.environment["GHM_UPDATE_FEED"]
    }

    func allowedChannels(for updater: SPUUpdater) -> Set<String> {
        guard let channel = ProcessInfo.processInfo.environment["GHM_UPDATE_CHANNEL"], !channel.isEmpty else {
            return []
        }
        return [channel]
    }
}

/// Sparkle integration: scheduled background checks plus the menu item.
@MainActor
final class UpdaterModel: NSObject, ObservableObject {
    private let delegate = SparkleDelegate()
    private let controller: SPUStandardUpdaterController

    override init() {
        controller = SPUStandardUpdaterController(
            startingUpdater: true,
            updaterDelegate: delegate,
            userDriverDelegate: nil
        )
        super.init()
    }

    func checkForUpdates() {
        controller.checkForUpdates(nil)
    }
}
