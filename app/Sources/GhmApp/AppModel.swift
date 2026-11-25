import Foundation
import SwiftUI

@MainActor
final class AppModel: ObservableObject {
    @Published private(set) var accounts: [Account] = []
    @Published private(set) var repos: [Repo] = []
    @Published private(set) var active = ""
    @Published private(set) var root = "~/work"
    @Published private(set) var loading = false
    @Published var selectedLogin = ""      // "" = all accounts
    @Published var search = ""
    @Published var errorMessage: String?
    @Published var actionMessage: String?
    @Published var runningAction = false
    @Published private(set) var lastRefresh: Date?

    private let cli = GhmCLI.shared

    var ghmBinaryPath: String { cli.binaryPath }

    var filteredRepos: [Repo] {
        let login = selectedLogin
        let query = search.trimmingCharacters(in: .whitespaces).lowercased()
        return repos
            .filter { login.isEmpty || $0.login.caseInsensitiveCompare(login) == .orderedSame }
            .filter {
                guard !query.isEmpty else { return true }
                return $0.displayName.lowercased().contains(query)
                    || $0.branch.lowercased().contains(query)
                    || $0.path.lowercased().contains(query)
            }
            .sorted { $0.displayName.localizedCaseInsensitiveCompare($1.displayName) == .orderedAscending }
    }

    var selectedAccount: Account? {
        accounts.first { $0.login == selectedLogin }
    }

    var dirtyCount: Int { filteredRepos.filter { $0.dirty > 0 }.count }
    var behindCount: Int { filteredRepos.filter { $0.behind > 0 }.count }

    var title: String {
        selectedAccount.map { $0.login } ?? "All repositories"
    }

    func account(for login: String) -> Account? {
        accounts.first { $0.login.caseInsensitiveCompare(login) == .orderedSame }
    }

    func repo(for path: String) -> Repo? {
        repos.first { $0.path == path }
    }

    func refresh(silently: Bool = false) async {
        if !silently { loading = true }
        defer { loading = false }
        do {
            let state = try await cli.state()
            accounts = state.accounts
            repos = state.repos
            active = state.active
            root = state.root
            lastRefresh = Date()
            if !selectedLogin.isEmpty && !state.accounts.contains(where: { $0.login == selectedLogin }) {
                selectedLogin = ""
            }
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func run(_ action: String, on repo: Repo) async {
        runningAction = true
        actionMessage = "\(action) \(repo.shortName)…"
        defer { runningAction = false }
        do {
            let result = try await cli.git(path: repo.path, action: action)
            if result.ok == true {
                actionMessage = "\(action) \(repo.shortName): ok"
                await refresh(silently: true)
            } else {
                actionMessage = nil
                errorMessage = result.error ?? "\(action) failed"
            }
        } catch {
            actionMessage = nil
            errorMessage = error.localizedDescription
        }
    }

    func openInBrowser(_ repo: Repo) {
        guard let url = repo.githubURL else {
            errorMessage = "This repository has no GitHub remote."
            return
        }
        NSWorkspace.shared.open(url)
    }

    func revealInFinder(_ repo: Repo) {
        NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: repo.path)])
    }

    func copyPath(_ repo: Repo) {
        NSPasteboard.general.clearContents()
        NSPasteboard.general.setString(repo.path, forType: .string)
        actionMessage = "Copied \(repo.tildePath)"
    }
}
