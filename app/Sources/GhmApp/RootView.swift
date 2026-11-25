import SwiftUI

struct RootView: View {
    @EnvironmentObject private var model: AppModel
    @EnvironmentObject private var updater: UpdaterModel
    @State private var showAdd = false
    @State private var showDoctor = false
    @State private var selectedPath: String?

    var body: some View {
        ZStack {
            LiquidBackground()

            HStack(spacing: 14) {
                SidebarView(showAdd: $showAdd, showDoctor: $showDoctor)
                    .frame(width: 272)
                RepoDetailView(
                    selectedPath: $selectedPath,
                    showAdd: $showAdd,
                    showDoctor: $showDoctor
                )
            }
            .padding(.init(top: 30, leading: 14, bottom: 14, trailing: 14))
        }
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
        .onChange(of: model.selectedLogin) {
            selectedPath = nil
        }
        .onReceive(NotificationCenter.default.publisher(for: .ghmShowAdd)) { _ in
            showAdd = true
        }
    }
}

// MARK: - Sidebar

private struct SidebarView: View {
    @EnvironmentObject private var model: AppModel
    @EnvironmentObject private var updater: UpdaterModel
    @Binding var showAdd: Bool
    @Binding var showDoctor: Bool
    @Namespace private var selection

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            header

            ScrollView {
                VStack(alignment: .leading, spacing: 6) {
                    Text("REPOSITORIES")
                        .font(.caption2.weight(.semibold))
                        .foregroundStyle(.tertiary)
                        .padding(.horizontal, 10)

                    allRow

                    ForEach(model.accounts) { account in
                        accountRow(account)
                    }
                }
                .padding(.vertical, 2)
            }
            .scrollIndicators(.hidden)

            Spacer(minLength: 0)

            manage

            footer
        }
        .padding(14)
        .glassPanel(28)
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .firstTextBaseline, spacing: 8) {
                Text("ghm")
                    .font(.system(size: 24, weight: .bold, design: .rounded))
                    .foregroundStyle(.white.opacity(0.92))
                Spacer()
                GlassBadge(text: version, systemImage: "shippingbox")
            }

            Text("One machine, every GitHub account")
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .padding(.horizontal, 4)
    }

    private var version: String {
        Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "dev"
    }

    private var allRow: some View {
        let selected = model.selectedLogin.isEmpty
        return Button {
            withAnimation(.smooth(duration: 0.22)) { model.selectedLogin = "" }
        } label: {
            HStack(spacing: 10) {
                Image(systemName: "square.stack.3d.up.fill")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(selected ? .white : Color.ghmAccent)
                    .frame(width: 26, height: 26)
                    .background(.white.opacity(selected ? 0.18 : 0.06), in: .circle)

                VStack(alignment: .leading, spacing: 1) {
                    Text("All repositories")
                        .font(.callout.weight(.medium))
                    Text("\(model.repos.count) checkouts")
                        .font(.caption2)
                        .foregroundStyle(.tertiary)
                }

                Spacer()

                if selected {
                    Image(systemName: "checkmark")
                        .font(.caption2.bold())
                        .foregroundStyle(.white.opacity(0.9))
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 7)
            .contentShape(.rect(cornerRadius: 14))
        }
        .buttonStyle(.plain)
        .modifier(SidebarSelection(selected: selected, tint: .ghmAccent))
        .glassEffectID("all", in: selection)
    }

    private func accountRow(_ account: Account) -> some View {
        let selected = model.selectedLogin == account.login
        return Button {
            withAnimation(.smooth(duration: 0.22)) { model.selectedLogin = account.login }
        } label: {
            HStack(spacing: 10) {
                Avatar(account: account)

                VStack(alignment: .leading, spacing: 1) {
                    Text(account.login)
                        .font(.callout.weight(.medium))
                        .lineLimit(1)
                    Text(account.name)
                        .font(.caption2)
                        .foregroundStyle(.tertiary)
                        .lineLimit(1)
                }

                Spacer(minLength: 4)

                HStack(spacing: 4) {
                    if !account.keyOK {
                        Image(systemName: "key.slash")
                            .foregroundStyle(Color.ghmWarning)
                    }
                    if !account.tokenOK {
                        Image(systemName: "keyboard.badge.ellipsis")
                            .foregroundStyle(Color.ghmDanger)
                    }
                    Text("\(account.repos)")
                        .font(.caption2.monospacedDigit())
                        .foregroundStyle(.tertiary)
                }
            }
            .padding(.horizontal, 10)
            .padding(.vertical, 7)
            .contentShape(.rect(cornerRadius: 14))
        }
        .buttonStyle(.plain)
        .modifier(SidebarSelection(selected: selected, tint: .ghmAccent))
        .glassEffectID(account.login, in: selection)
    }

    private var manage: some View {
        VStack(spacing: 6) {
            Button {
                showAdd = true
            } label: {
                Label("Add account", systemImage: "person.badge.plus")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.glassProminent)
            .keyboardShortcut("n", modifiers: [.command, .shift])
            .help("Add a GitHub account")

            HStack(spacing: 6) {
                Button {
                    showDoctor = true
                } label: {
                    Label("Health", systemImage: "stethoscope")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.glass)
                .help("Run ghm doctor")

                Button {
                    Task { await model.refresh() }
                } label: {
                    Label("Refresh", systemImage: "arrow.clockwise")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.glass)
                .keyboardShortcut("r")
                .disabled(model.loading)
            }

            Button {
                updater.checkForUpdates()
            } label: {
                Label("Check for updates", systemImage: "arrow.triangle.2.circlepath")
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.glass)
            .help("Sparkle update check")
        }
        .controlSize(.small)
    }

    private var footer: some View {
        VStack(alignment: .leading, spacing: 3) {
            HStack(spacing: 6) {
                Circle()
                    .fill(model.active.isEmpty ? Color.ghmWarning : Color.ghmPositive)
                    .frame(width: 6, height: 6)
                Text(model.active.isEmpty ? "gh: no active login" : "gh active: \(model.active)")
                    .lineLimit(1)
            }
            Text(model.root)
                .font(.caption2.monospaced())
                .foregroundStyle(.tertiary)
                .lineLimit(1)
                .truncationMode(.middle)
        }
        .font(.caption2)
        .foregroundStyle(.secondary)
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .glassPanel(14)
    }
}

private struct SidebarSelection: ViewModifier {
    let selected: Bool
    let tint: Color

    func body(content: Content) -> some View {
        content.glassEffect(
            selected ? .regular.tint(tint.opacity(0.16)).interactive() : .identity,
            in: .rect(cornerRadius: 14)
        )
    }
}

// MARK: - Detail

private struct RepoDetailView: View {
    @EnvironmentObject private var model: AppModel
    @Binding var selectedPath: String?
    @Binding var showAdd: Bool
    @Binding var showDoctor: Bool
    @State private var hovered: String?

    private let accountWidth: CGFloat = 126
    private let branchWidth: CGFloat = 150
    private let numberWidth: CGFloat = 38
    private let stateWidth: CGFloat = 92

    var body: some View {
        VStack(spacing: 10) {
            header

            if model.loading && model.repos.isEmpty {
                loading
            } else if model.filteredRepos.isEmpty {
                empty
            } else {
                list
            }

            statusBar
        }
        .padding(14)
        .glassPanel(28)
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .center, spacing: 10) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(model.title)
                        .font(.system(size: 22, weight: .semibold, design: .rounded))
                    Text(subtitle)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                }

                Spacer()

                if model.dirtyCount > 0 {
                    GlassBadge(text: "\(model.dirtyCount) dirty", systemImage: "pencil.line", tint: .ghmWarning)
                }
                if model.behindCount > 0 {
                    GlassBadge(text: "\(model.behindCount) behind", systemImage: "arrow.down", tint: .ghmInfo)
                }

                GlassIconButton(systemImage: "arrow.clockwise", help: "Rescan (⌘R)") {
                    Task { await model.refresh() }
                }
                GlassIconButton(systemImage: "stethoscope", help: "Health checks") {
                    showDoctor = true
                }
                Button {
                    showAdd = true
                } label: {
                    Label("Add account", systemImage: "person.badge.plus")
                }
                .buttonStyle(.glassProminent)
                .controlSize(.small)
            }

            HStack(spacing: 10) {
                HStack(spacing: 6) {
                    Image(systemName: "magnifyingglass")
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                    TextField("Filter by repository, branch or path", text: $model.search)
                        .textFieldStyle(.plain)
                        .font(.callout)
                    if !model.search.isEmpty {
                        Button {
                            model.search = ""
                        } label: {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundStyle(.tertiary)
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.horizontal, 11)
                .padding(.vertical, 7)
                .glassPanel(12)

                Spacer()

                GlassBadge(text: "\(model.filteredRepos.count) repos", systemImage: "folder")
            }
        }
    }

    private var subtitle: String {
        if let account = model.selectedAccount {
            return "\(account.name) · \(account.email)"
        }
        let logins = model.accounts.map(\.login).joined(separator: ", ")
        return logins.isEmpty ? "No accounts configured yet" : logins
    }

    private var loading: some View {
        VStack(spacing: 14) {
            ProgressView().controlSize(.large)
            Text("Scanning repositories…")
                .font(.callout)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .glassPanel(20)
    }

    private var empty: some View {
        VStack(spacing: 12) {
            Image(systemName: model.search.isEmpty ? "tray" : "magnifyingglass")
                .font(.system(size: 36, weight: .light))
                .foregroundStyle(.secondary)
            Text(model.search.isEmpty ? "No repositories for this account" : "No match for “\(model.search)”")
                .font(.headline)
            Text(model.search.isEmpty
                 ? "Clone with ghm clone owner/repo, or migrate with ghm adopt."
                 : "Try a shorter filter.")
                .font(.callout)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .glassPanel(20)
    }

    private var list: some View {
        VStack(spacing: 0) {
            headerRow
                .padding(.horizontal, 14)
                .padding(.vertical, 8)

            Divider().overlay(.white.opacity(0.08))

            ScrollView {
                LazyVStack(spacing: 4) {
                    ForEach(model.filteredRepos) { repo in
                        RepoRow(
                            repo: repo,
                            selected: selectedPath == repo.path,
                            hovered: hovered == repo.path,
                            accountWidth: accountWidth,
                            branchWidth: branchWidth,
                            numberWidth: numberWidth,
                            stateWidth: stateWidth
                        )
                        .onHover { inside in
                            if inside {
                                hovered = repo.path
                            } else if hovered == repo.path {
                                hovered = nil
                            }
                        }
                        .onTapGesture {
                            withAnimation(.smooth(duration: 0.15)) {
                                selectedPath = repo.path
                            }
                        }
                        .simultaneousGesture(TapGesture(count: 2).onEnded {
                            model.openInBrowser(repo)
                        })
                        .contextMenu {
                            RepoActionMenu(repo: repo)
                        }
                    }
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 8)
            }
        }
        .background(.black.opacity(0.12), in: .rect(cornerRadius: 18))
        .overlay(
            RoundedRectangle(cornerRadius: 18)
                .strokeBorder(.white.opacity(0.06), lineWidth: 0.5)
        )
    }

    private var headerRow: some View {
        HStack(spacing: 10) {
            Text("REPOSITORY")
                .frame(maxWidth: .infinity, alignment: .leading)
            Text("ACCOUNT").frame(width: accountWidth, alignment: .leading)
            Text("BRANCH").frame(width: branchWidth, alignment: .leading)
            Text("↑").frame(width: numberWidth, alignment: .trailing)
            Text("↓").frame(width: numberWidth, alignment: .trailing)
            Text("STATE").frame(width: stateWidth, alignment: .trailing)
        }
        .font(.caption2.weight(.semibold))
        .foregroundStyle(.tertiary)
    }

    private var statusBar: some View {
        HStack(spacing: 10) {
            if let repo = selectedPath.flatMap({ model.repo(for: $0) }) {
                Image(systemName: "folder.fill")
                    .font(.caption2)
                    .foregroundStyle(Color.ghmInfo)
                Text(repo.tildePath)
                    .font(.caption.monospaced())
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                    .truncationMode(.middle)

                Spacer()

                Button("Fetch") { Task { await model.run("fetch", on: repo) } }
                Button("Pull") { Task { await model.run("pull", on: repo) } }
                Button("Push") { Task { await model.run("push", on: repo) } }
                GlassIconButton(systemImage: "arrow.up.right.square", help: "Open on GitHub") {
                    model.openInBrowser(repo)
                }
                .disabled(repo.githubURL == nil)
            } else {
                Text("\(model.filteredRepos.count) repositories")
                    .foregroundStyle(.secondary)
                Text("\(model.dirtyCount) dirty")
                    .foregroundStyle(model.dirtyCount > 0 ? Color.ghmWarning : Color.secondary)
                Text("\(model.behindCount) behind")
                    .foregroundStyle(model.behindCount > 0 ? Color.ghmInfo : Color.secondary)

                Spacer()

                if model.runningAction {
                    ProgressView().controlSize(.small)
                }
                if let message = model.actionMessage {
                    Text(message)
                        .foregroundStyle(.secondary)
                        .lineLimit(1)
                        .transition(.opacity)
                }
                if let last = model.lastRefresh {
                    Text(last, style: .time)
                        .foregroundStyle(.tertiary)
                }
            }
        }
        .font(.caption)
        .buttonStyle(.glass)
        .controlSize(.small)
        .padding(.horizontal, 12)
        .padding(.vertical, 7)
        .glassPanel(14)
        .animation(.smooth(duration: 0.2), value: model.actionMessage)
        .animation(.smooth(duration: 0.2), value: selectedPath)
    }
}

// MARK: - Rows

private struct RepoRow: View {
    @EnvironmentObject private var model: AppModel
    let repo: Repo
    let selected: Bool
    let hovered: Bool
    let accountWidth: CGFloat
    let branchWidth: CGFloat
    let numberWidth: CGFloat
    let stateWidth: CGFloat

    var body: some View {
        HStack(spacing: 10) {
            HStack(spacing: 8) {
                Image(systemName: repo.isGitHub ? "chevron.left.forwardslash.chevron.right" : "shippingbox")
                    .font(.caption)
                    .foregroundStyle(selected ? .white.opacity(0.9) : .secondary)
                    .frame(width: 16)
                VStack(alignment: .leading, spacing: 1) {
                    Text(repo.displayName)
                        .font(.callout.weight(.medium))
                        .lineLimit(1)
                    Text(repo.tildePath)
                        .font(.caption2.monospaced())
                        .foregroundStyle(.tertiary)
                        .lineLimit(1)
                        .truncationMode(.middle)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)

            HStack(spacing: 6) {
                if let account = model.account(for: repo.login) {
                    Avatar(account: account, size: 18)
                }
                Text(repo.login.isEmpty ? "—" : repo.login)
                    .lineLimit(1)
            }
            .frame(width: accountWidth, alignment: .leading)
            .foregroundStyle(.secondary)

            Text(repo.branch.isEmpty ? "—" : repo.branch)
                .font(.caption.monospaced())
                .foregroundStyle(.secondary)
                .lineLimit(1)
                .frame(width: branchWidth, alignment: .leading)

            Text("\(repo.ahead)")
                .font(.caption.monospacedDigit())
                .foregroundStyle(repo.ahead > 0 ? Color.ghmPositive : Color.secondary)
                .frame(width: numberWidth, alignment: .trailing)

            Text("\(repo.behind)")
                .font(.caption.monospacedDigit())
                .foregroundStyle(repo.behind > 0 ? Color.ghmInfo : Color.secondary)
                .frame(width: numberWidth, alignment: .trailing)

            StatePill(repo: repo)
                .frame(width: stateWidth, alignment: .trailing)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 7)
        .contentShape(.rect(cornerRadius: 12))
        .background(
            selected ? AnyShapeStyle(.white.opacity(0.10)) : AnyShapeStyle(hovered ? .white.opacity(0.05) : .clear),
            in: .rect(cornerRadius: 12)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .strokeBorder(selected ? .white.opacity(0.22) : .clear, lineWidth: 0.8)
        )
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
