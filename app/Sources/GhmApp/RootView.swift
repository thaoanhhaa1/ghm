import SwiftUI

struct RootView: View {
    @EnvironmentObject private var model: AppModel
    @State private var showAdd = false
    @State private var showDoctor = false
    @State private var selection = Set<String>()

    var body: some View {
        NavigationSplitView {
            SidebarView(showAdd: $showAdd, showDoctor: $showDoctor)
                .navigationSplitViewColumnWidth(min: 230, ideal: 260, max: 320)
        } detail: {
            RepoListView(selection: $selection, showAdd: $showAdd, showDoctor: $showDoctor)
        }
        .navigationTitle("ghm")
        .sheet(isPresented: $showAdd) {
            AddAccountView()
                .environmentObject(model)
        }
        .sheet(isPresented: $showDoctor) {
            DoctorView()
        }
        .alert("ghm", isPresented: Binding(
            get: { model.errorMessage != nil },
            set: { if !$0 { model.errorMessage = nil } }
        )) {
            Button("OK", role: .cancel) { model.errorMessage = nil }
        } message: {
            Text(model.errorMessage ?? "")
        }
        .task { await model.refresh() }
    }
}

// MARK: - Sidebar

private struct SidebarView: View {
    @EnvironmentObject private var model: AppModel
    @Binding var showAdd: Bool
    @Binding var showDoctor: Bool

    var body: some View {
        List(selection: $model.selectedLogin) {
            Section("Repositories") {
                Label("All repositories", systemImage: "square.stack.3d.up")
                    .badge(model.repos.count)
                    .tag("")
                ForEach(model.accounts) { account in
                    AccountRow(account: account)
                        .tag(account.login)
                }
            }

            Section("Manage") {
                Button {
                    showAdd = true
                } label: {
                    Label("Add account…", systemImage: "person.badge.plus")
                }
                .buttonStyle(.plain)

                Button {
                    showDoctor = true
                } label: {
                    Label("Health checks…", systemImage: "stethoscope")
                }
                .buttonStyle(.plain)

                Button {
                    Task { await model.refresh() }
                } label: {
                    Label("Refresh", systemImage: "arrow.clockwise")
                }
                .buttonStyle(.plain)
            }
        }
        .listStyle(.sidebar)
        .safeAreaInset(edge: .bottom) {
            VStack(alignment: .leading, spacing: 2) {
                Divider()
                HStack(spacing: 6) {
                    Image(systemName: "checkmark.seal.fill")
                        .foregroundStyle(.green)
                    Text("gh active: \(model.active.isEmpty ? "none" : model.active)")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Text("root: \(model.root)")
                    .font(.caption2.monospaced())
                    .foregroundStyle(.tertiary)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(.bar)
        }
    }
}

private struct AccountRow: View {
    let account: Account

    var body: some View {
        HStack(spacing: 8) {
            ZStack {
                Circle()
                    .fill(LinearGradient(
                        colors: [.accentColor.opacity(0.85), .accentColor.opacity(0.4)],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    ))
                Text(account.initials)
                    .font(.caption2.bold())
                    .foregroundStyle(.white)
            }
            .frame(width: 22, height: 22)

            VStack(alignment: .leading, spacing: 1) {
                Text(account.login)
                    .font(.body.weight(.medium))
                Text(account.name)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }

            Spacer()

            if !account.keyOK {
                Image(systemName: "key.slash")
                    .foregroundStyle(.orange)
                    .help("SSH key missing")
            }
            if !account.tokenOK {
                Image(systemName: "keyboard.badge.ellipsis")
                    .foregroundStyle(.red)
                    .help("gh token missing")
            }
            Text("\(account.repos)")
                .font(.caption.monospacedDigit())
                .foregroundStyle(.secondary)
        }
        .padding(.vertical, 2)
    }
}

// MARK: - Repo list

private struct RepoListView: View {
    @EnvironmentObject private var model: AppModel
    @Binding var selection: Set<String>
    @Binding var showAdd: Bool
    @Binding var showDoctor: Bool

    var body: some View {
        VStack(spacing: 0) {
            if model.loading && model.repos.isEmpty {
                VStack(spacing: 12) {
                    ProgressView()
                    Text("Scanning repositories…")
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if model.filteredRepos.isEmpty {
                emptyState
            } else {
                RepoTable(repos: model.filteredRepos, selection: $selection)
            }
            Divider()
            StatusBar(selection: selection)
        }
        .navigationTitle(model.title)
        .searchable(text: $model.search, placement: .toolbar, prompt: "Filter repositories")
        .toolbar {
            ToolbarItemGroup(placement: .primaryAction) {
                Button {
                    Task { await model.refresh() }
                } label: {
                    Label("Refresh", systemImage: "arrow.clockwise")
                }
                .help("Rescan repositories (⌘R)")
                .disabled(model.loading)

                Button {
                    showDoctor = true
                } label: {
                    Label("Health checks", systemImage: "stethoscope")
                }
                .help("Run ghm doctor")

                Button {
                    showAdd = true
                } label: {
                    Label("Add account", systemImage: "person.badge.plus")
                }
                .help("Add a GitHub account")
            }
        }
    }

    private var emptyState: some View {
        VStack(spacing: 10) {
            Image(systemName: "tray")
                .font(.system(size: 34))
                .foregroundStyle(.tertiary)
            Text(model.search.isEmpty ? "No repositories for this account" : "No match for “\(model.search)”")
                .font(.headline)
            Text("Clone with ghm clone owner/repo, or migrate with ghm adopt.")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

}

// MARK: - Table

private struct RepoTable: View {
    @EnvironmentObject private var model: AppModel
    let repos: [Repo]
    @Binding var selection: Set<String>

    var body: some View {
        table
            .contextMenu(forSelectionType: String.self) { ids in
                if let repo = selectedRepo(ids) {
                    RepoActionMenu(repo: repo)
                }
            } primaryAction: { ids in
                if let repo = selectedRepo(ids) {
                    model.openInBrowser(repo)
                }
            }
    }

    private var table: some View {
        Table(repos, selection: $selection) {
            TableColumn("Repository") { (repo: Repo) in
                RepoCell(repo: repo)
            }
            .width(min: 220, ideal: 320)

            TableColumn("Account") { (repo: Repo) in
                AccountCell(repo: repo)
            }
            .width(min: 90, ideal: 130)

            TableColumn("Branch") { (repo: Repo) in
                BranchCell(repo: repo)
            }
            .width(min: 90, ideal: 150)

            TableColumn("\u{2191}") { (repo: Repo) in
                AheadCell(repo: repo)
            }
            .width(36)

            TableColumn("\u{2193}") { (repo: Repo) in
                BehindCell(repo: repo)
            }
            .width(36)

            TableColumn("State") { (repo: Repo) in
                StateBadge(repo: repo)
            }
            .width(min: 80, ideal: 100)
        }
    }

    private func selectedRepo(_ ids: Set<String>) -> Repo? {
        guard let id = ids.first else { return nil }
        return model.repo(for: id)
    }
}

private struct RepoActionMenu: View {
    @EnvironmentObject private var model: AppModel
    let repo: Repo

    var body: some View {
        Button("Fetch") { Task { await model.run("fetch", on: repo) } }
        Button("Pull") { Task { await model.run("pull", on: repo) } }
        Button("Push") { Task { await model.run("push", on: repo) } }
        Divider()
        Button("Open on GitHub") { model.openInBrowser(repo) }
            .disabled(repo.githubURL == nil)
        Button("Reveal in Finder") { model.revealInFinder(repo) }
        Button("Copy path") { model.copyPath(repo) }
    }
}

private struct AccountCell: View {
    let repo: Repo

    var body: some View {
        Text(repo.login.isEmpty ? "\u{2014}" : repo.login)
            .foregroundStyle(.secondary)
    }
}

private struct BranchCell: View {
    let repo: Repo

    var body: some View {
        Text(repo.branch.isEmpty ? "\u{2014}" : repo.branch)
            .font(.body.monospaced())
            .foregroundStyle(.secondary)
    }
}

private struct AheadCell: View {
    let repo: Repo

    var body: some View {
        Text("\(repo.ahead)")
            .font(.body.monospacedDigit())
            .foregroundStyle(repo.ahead > 0 ? Color.primary : Color.secondary)
    }
}

private struct BehindCell: View {
    let repo: Repo

    var body: some View {
        Text("\(repo.behind)")
            .font(.body.monospacedDigit())
            .foregroundStyle(repo.behind > 0 ? Color.orange : Color.secondary)
    }
}

private struct RepoCell: View {
    let repo: Repo

    var body: some View {
        HStack(spacing: 8) {
            Image(systemName: repo.isGitHub ? "chevron.left.forwardslash.chevron.right" : "shippingbox")
                .foregroundStyle(.tertiary)
            VStack(alignment: .leading, spacing: 1) {
                Text(repo.displayName)
                    .lineLimit(1)
                Text(repo.tildePath)
                    .font(.caption2.monospaced())
                    .foregroundStyle(.tertiary)
                    .lineLimit(1)
                    .truncationMode(.middle)
            }
        }
        .padding(.vertical, 1)
    }
}

private struct StateBadge: View {
    let repo: Repo

    var body: some View {
        Text(repo.stateLabel)
            .font(.caption2.weight(.medium))
            .padding(.horizontal, 7)
            .padding(.vertical, 2)
            .background(background, in: Capsule())
            .foregroundStyle(foreground)
    }

    private var background: Color {
        if repo.dirty > 0 { return .orange.opacity(0.18) }
        if repo.behind > 0 { return .blue.opacity(0.16) }
        if repo.ahead > 0 { return .green.opacity(0.16) }
        return .secondary.opacity(0.12)
    }

    private var foreground: Color {
        if repo.dirty > 0 { return .orange }
        if repo.behind > 0 { return .blue }
        if repo.ahead > 0 { return .green }
        return .secondary
    }
}

// MARK: - Status bar

private struct StatusBar: View {
    @EnvironmentObject private var model: AppModel
    let selection: Set<String>

    var body: some View {
        HStack(spacing: 12) {
            if let repo = model.repo(for: selection.first ?? "") {
                Text(repo.tildePath)
                    .font(.caption.monospaced())
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                    .truncationMode(.middle)
                Spacer()
                Button("Fetch") { Task { await model.run("fetch", on: repo) } }
                Button("Pull") { Task { await model.run("pull", on: repo) } }
                Button("Push") { Task { await model.run("push", on: repo) } }
                Button {
                    model.openInBrowser(repo)
                } label: {
                    Image(systemName: "arrow.up.right.square")
                }
                .disabled(repo.githubURL == nil)
            } else {
                Text("\(model.filteredRepos.count) repositories")
                Text("\(model.dirtyCount) dirty")
                    .foregroundStyle(model.dirtyCount > 0 ? .orange : .secondary)
                Text("\(model.behindCount) behind")
                    .foregroundStyle(model.behindCount > 0 ? .blue : .secondary)
                Spacer()
                if model.runningAction {
                    ProgressView().controlSize(.small)
                }
                if let message = model.actionMessage {
                    Text(message)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }
                if let last = model.lastRefresh {
                    Text(last, style: .time)
                        .foregroundStyle(.tertiary)
                }
            }
        }
        .font(.caption)
        .padding(.horizontal, 12)
        .padding(.vertical, 6)
    }
}
