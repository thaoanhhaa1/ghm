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

            VStack(spacing: 12) {
                TopBar(showAdd: $showAdd, showDoctor: $showDoctor)

                HStack(spacing: 12) {
                    AccountsPanel(showAdd: $showAdd, showDoctor: $showDoctor)
                        .frame(width: 278)
                    ReposPanel(selectedPath: $selectedPath)
                }

                BottomBar(selectedPath: $selectedPath, showAdd: $showAdd, showDoctor: $showDoctor)
            }
            .padding(.horizontal, 16)
            .padding(.top, 40)
            .padding(.bottom, 14)
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

// MARK: - Top bar (navigation bar)

private struct TopBar: View {
    @EnvironmentObject private var model: AppModel
    @Binding var showAdd: Bool
    @Binding var showDoctor: Bool

    var body: some View {
        HStack(spacing: 12) {
            HStack(spacing: 9) {
                Image(systemName: "arrow.triangle.branch")
                    .font(.system(size: 13, weight: .bold))
                    .foregroundStyle(.white)
                    .frame(width: 27, height: 27)
                    .background(
                        LinearGradient(
                            colors: [.iosBlue, .iosIndigo],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        ),
                        in: .rect(cornerRadius: 9)
                    )
                    .overlay(
                        RoundedRectangle(cornerRadius: 9)
                            .strokeBorder(.white.opacity(0.35), lineWidth: 0.7)
                    )
                Text("ghm")
                    .font(.system(size: 17, weight: .semibold, design: .rounded))
                GlassBadge(text: version, systemImage: "shippingbox")
            }

            search
                .frame(minWidth: 260, idealWidth: 380, maxWidth: 520)

            Spacer(minLength: 8)

            if model.dirtyCount > 0 {
                GlassBadge(text: "\(model.dirtyCount)", systemImage: "pencil.line", tint: .ghmWarning)
            }
            if model.behindCount > 0 {
                GlassBadge(text: "\(model.behindCount)", systemImage: "arrow.down", tint: .ghmInfo)
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
                    .labelStyle(.titleAndIcon)
            }
            .buttonStyle(.glassProminent)
            .buttonBorderShape(.capsule)
            .tint(.iosBlue)
            .controlSize(.regular)
            .keyboardShortcut("n", modifiers: [.command, .shift])
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
        .glassPanel(24)
    }

    private var version: String {
        Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "dev"
    }

    private var search: some View {
        HStack(spacing: 7) {
            Image(systemName: "magnifyingglass")
                .font(.callout)
                .foregroundStyle(.tertiary)
            TextField("Search repositories", text: $model.search)
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
        .padding(.horizontal, 13)
        .padding(.vertical, 6)
        .background(.white.opacity(0.10), in: .capsule)
        .overlay(Capsule().strokeBorder(.white.opacity(0.10), lineWidth: 0.5))
    }
}

// MARK: - Accounts (sidebar island)

private struct AccountsPanel: View {
    @EnvironmentObject private var model: AppModel
    @EnvironmentObject private var updater: UpdaterModel
    @Binding var showAdd: Bool
    @Binding var showDoctor: Bool
    @Namespace private var lens

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(spacing: 6) {
                Text("Accounts")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(.secondary)
                Spacer()
                Text("\(model.accounts.count)")
                    .font(.caption.monospacedDigit())
                    .foregroundStyle(.tertiary)
                GlassIconButton(systemImage: "arrow.triangle.2.circlepath", help: "Check for updates") {
                    updater.checkForUpdates()
                }
            }
            .padding(.leading, 6)

            ScrollView {
                GlassEffectContainer(spacing: 8) {
                    VStack(spacing: 4) {
                        allRow
                        ForEach(model.accounts) { account in
                            accountRow(account)
                        }
                    }
                    .padding(.vertical, 2)
                }
            }
            .scrollIndicators(.hidden)

            Spacer(minLength: 0)

            VStack(spacing: 8) {
                Button {
                    showAdd = true
                } label: {
                    Label("Add account", systemImage: "person.badge.plus")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.glassProminent)
                .buttonBorderShape(.capsule)
                .tint(.iosBlue)
                .keyboardShortcut("n", modifiers: [.command, .shift])

                HStack(spacing: 8) {
                    Button {
                        showDoctor = true
                    } label: {
                        Label("Health", systemImage: "stethoscope")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.glass)
                    .buttonBorderShape(.capsule)

                    Button {
                        Task { await model.refresh() }
                    } label: {
                        Label("Refresh", systemImage: "arrow.clockwise")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.glass)
                    .buttonBorderShape(.capsule)
                    .keyboardShortcut("r")
                    .disabled(model.loading)
                }
            }
            .controlSize(.small)

            footer
        }
        .padding(14)
        .glassPanel(30)
    }

    private var allRow: some View {
        let selected = model.selectedLogin.isEmpty
        return Button {
            withAnimation(.smooth(duration: 0.28, extraBounce: 0.06)) {
                model.selectedLogin = ""
            }
        } label: {
            HStack(spacing: 10) {
                Image(systemName: "square.stack.3d.up.fill")
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(width: 28, height: 28)
                    .background(
                        LinearGradient(
                            colors: selected ? [.iosBlue, .iosIndigo] : [.white.opacity(0.16), .white.opacity(0.08)],
                            startPoint: .topLeading,
                            endPoint: .bottomTrailing
                        ),
                        in: .circle
                    )

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
                        .foregroundStyle(.white)
                }
            }
            .padding(.horizontal, 9)
            .padding(.vertical, 7)
            .contentShape(.rect(cornerRadius: 16))
        }
        .buttonStyle(.plain)
        .modifier(LensSelection(selected: selected))
        .glassEffectID("all", in: lens)
    }

    private func accountRow(_ account: Account) -> some View {
        let selected = model.selectedLogin == account.login
        return Button {
            withAnimation(.smooth(duration: 0.28, extraBounce: 0.06)) {
                model.selectedLogin = account.login
            }
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

                HStack(spacing: 5) {
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
            .padding(.horizontal, 9)
            .padding(.vertical, 7)
            .contentShape(.rect(cornerRadius: 16))
        }
        .buttonStyle(.plain)
        .modifier(LensSelection(selected: selected))
        .glassEffectID(account.login, in: lens)
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
        .glassEffect(.regular, in: .rect(cornerRadius: 16))
    }
}

/// The iOS 26 "lens": the selected item gets its own piece of glass, which
/// morphs between rows through the shared namespace.
private struct LensSelection: ViewModifier {
    let selected: Bool

    func body(content: Content) -> some View {
        content.glassEffect(
            selected ? .regular.tint(.white.opacity(0.14)).interactive() : .identity,
            in: .rect(cornerRadius: 16)
        )
    }
}

// MARK: - Repositories (content island)

private struct ReposPanel: View {
    @EnvironmentObject private var model: AppModel
    @Binding var selectedPath: String?
    @State private var hovered: String?

    private let accountWidth: CGFloat = 128
    private let branchWidth: CGFloat = 150
    private let numberWidth: CGFloat = 36
    private let stateWidth: CGFloat = 92

    var body: some View {
        VStack(spacing: 10) {
            header

            Divider().overlay(.white.opacity(0.08))

            if model.loading && model.repos.isEmpty {
                loading
            } else if model.filteredRepos.isEmpty {
                empty
            } else {
                list
            }
        }
        .padding(16)
        .glassPanel(30)
    }

    private var header: some View {
        HStack(alignment: .firstTextBaseline, spacing: 10) {
            VStack(alignment: .leading, spacing: 2) {
                Text(model.title)
                    .font(.system(size: 24, weight: .semibold, design: .rounded))
                Text(subtitle)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }

            Spacer()

            GlassBadge(text: "\(model.filteredRepos.count) repos", systemImage: "folder")
            if !model.search.isEmpty {
                GlassBadge(text: "“\(model.search)”", systemImage: "line.3.horizontal.decrease")
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
    }

    private var empty: some View {
        VStack(spacing: 12) {
            Image(systemName: model.search.isEmpty ? "tray" : "magnifyingglass")
                .font(.system(size: 38, weight: .light))
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
    }

    private var list: some View {
        VStack(spacing: 0) {
            headerRow
                .padding(.horizontal, 14)
                .padding(.bottom, 6)

            ScrollView {
                LazyVStack(spacing: 2) {
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
                            withAnimation(.smooth(duration: 0.16)) {
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
                .padding(.vertical, 4)
            }
        }
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
        .font(.system(size: 10, weight: .semibold))
        .foregroundStyle(.tertiary)
    }
}

// MARK: - Bottom bar (tab bar pattern)

private struct BottomBar: View {
    @EnvironmentObject private var model: AppModel
    @Binding var selectedPath: String?
    @Binding var showAdd: Bool
    @Binding var showDoctor: Bool

    private var repo: Repo? {
        selectedPath.flatMap { model.repo(for: $0) }
    }

    var body: some View {
        HStack(spacing: 12) {
            HStack(spacing: 7) {
                Image(systemName: repo == nil ? "point.3.connected.trianglepath.dotted" : "folder.fill")
                    .font(.caption)
                    .foregroundStyle(repo == nil ? Color.secondary : Color.ghmInfo)
                VStack(alignment: .leading, spacing: 0) {
                    Text(repo?.tildePath ?? "\(model.filteredRepos.count) repositories")
                        .font(.caption.monospaced())
                        .foregroundStyle(repo == nil ? Color.secondary : Color.primary)
                        .lineLimit(1)
                        .truncationMode(.middle)
                    Text(repoSubtitle)
                        .font(.system(size: 10))
                        .foregroundStyle(.tertiary)
                        .lineLimit(1)
                }
                if model.runningAction {
                    ProgressView().controlSize(.small)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.leading, 6)

            if let message = model.actionMessage {
                Text(message)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
                    .transition(.opacity)
            }

            Divider().frame(height: 26).overlay(.white.opacity(0.12))

            HStack(spacing: 2) {
                ActionBarItem(systemImage: "arrow.down.circle", title: "Fetch", enabled: repo != nil) {
                    if let repo { Task { await model.run("fetch", on: repo) } }
                }
                ActionBarItem(systemImage: "arrow.down.to.line", title: "Pull", enabled: repo != nil) {
                    if let repo { Task { await model.run("pull", on: repo) } }
                }
                ActionBarItem(systemImage: "arrow.up.to.line", title: "Push", enabled: repo != nil) {
                    if let repo { Task { await model.run("push", on: repo) } }
                }
                ActionBarItem(
                    systemImage: "arrow.up.right.square",
                    title: "GitHub",
                    enabled: repo?.githubURL != nil
                ) {
                    if let repo { model.openInBrowser(repo) }
                }
                ActionBarItem(systemImage: "stethoscope", title: "Health") {
                    showDoctor = true
                }
            }

            Divider().frame(height: 26).overlay(.white.opacity(0.12))

            Button {
                showAdd = true
            } label: {
                Image(systemName: "plus")
                    .font(.system(size: 14, weight: .bold))
                    .frame(width: 22, height: 22)
            }
            .buttonStyle(.glassProminent)
            .buttonBorderShape(.circle)
            .tint(.iosBlue)
            .help("Add account (⇧⌘N)")
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .glassPanel(26)
        .animation(.smooth(duration: 0.2), value: model.actionMessage)
        .animation(.smooth(duration: 0.2), value: selectedPath)
    }

    private var repoSubtitle: String {
        guard let repo else {
            return "\(model.dirtyCount) dirty · \(model.behindCount) behind"
        }
        return [repo.login, repo.branch].filter { !$0.isEmpty }.joined(separator: " · ")
            + (repo.ahead > 0 ? " · ↑\(repo.ahead)" : "")
            + (repo.behind > 0 ? " · ↓\(repo.behind)" : "")
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
                    .foregroundStyle(selected ? AnyShapeStyle(Color.white.opacity(0.9)) : AnyShapeStyle(Color.secondary))
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
        .padding(.horizontal, 12)
        .padding(.vertical, 7)
        .contentShape(.rect(cornerRadius: 14))
        .background(
            selected
                ? AnyShapeStyle(.white.opacity(0.13))
                : AnyShapeStyle(hovered ? .white.opacity(0.06) : .clear),
            in: .rect(cornerRadius: 14)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 14)
                .strokeBorder(selected ? .white.opacity(0.20) : .clear, lineWidth: 0.8)
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
