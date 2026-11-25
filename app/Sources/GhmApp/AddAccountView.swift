import SwiftUI

struct AddAccountView: View {
    @EnvironmentObject private var model: AppModel
    @Environment(\.dismiss) private var dismiss
    @AppStorage("oauthClientID") private var clientID = "Iv1.b507a08c87ecfe98"

    private enum Method: String, CaseIterable {
        case browser = "Browser"
        case token = "Token"
    }

    @State private var method: Method = .browser
    @State private var token = ""
    @State private var busy = false
    @State private var device: DeviceCode?
    @State private var status = ""
    @State private var result: APIResult?
    @State private var failure: String?
    @State private var showAdvanced = false
    @State private var pollTask: Task<Void, Never>?

    var body: some View {
        VStack(alignment: .leading, spacing: 16) {
            header

            if result != nil {
                resultView
            } else if let failure {
                failureView(failure)
            } else if let device {
                deviceView(device)
            } else {
                Picker("Method", selection: $method) {
                    ForEach(Method.allCases, id: \.self) { Text($0.rawValue) }
                }
                .pickerStyle(.segmented)
                .labelsHidden()
                .disabled(busy)

                if method == .browser {
                    browserForm
                } else {
                    tokenForm
                }
            }

            Spacer(minLength: 0)
            footer
        }
        .padding(24)
        .frame(width: 580, height: 470)
        .onDisappear { pollTask?.cancel() }
    }

    private var header: some View {
        HStack(spacing: 10) {
            Image(systemName: "person.badge.plus")
                .font(.title2)
                .foregroundStyle(.tint)
            VStack(alignment: .leading, spacing: 2) {
                Text("Add GitHub account")
                    .font(.title2.bold())
                Text("ghm creates the SSH key, host alias, git config and folder.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
    }

    // MARK: Browser / device flow

    private var browserForm: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Sign in through github.com. No terminal needed: the app shows a code, opens the browser, and waits for approval.")
                .font(.callout)
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)

            Label("Requested scopes: repo, read:org, admin:public_key, workflow, gist", systemImage: "lock.shield")
                .font(.caption)
                .foregroundStyle(.secondary)

            if busy {
                HStack(spacing: 8) {
                    ProgressView().controlSize(.small)
                    Text("Starting device login…").font(.callout)
                }
            }

            DisclosureGroup("OAuth client id", isExpanded: $showAdvanced) {
                TextField("Iv1.…", text: $clientID)
                    .textFieldStyle(.roundedBorder)
                    .font(.body.monospaced())
                Text("Defaults to the public GitHub CLI OAuth app so it works out of the box. You can create your own OAuth App with Device Flow enabled and paste its client id here.")
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
                    .fixedSize(horizontal: false, vertical: true)
            }
            .font(.caption)
        }
    }

    private func deviceView(_ device: DeviceCode) -> some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("Enter this code on GitHub")
                .font(.callout.weight(.medium))

            HStack(spacing: 12) {
                Text(device.userCode)
                    .font(.system(size: 30, weight: .semibold, design: .monospaced))
                    .textSelection(.enabled)
                    .padding(.horizontal, 16)
                    .padding(.vertical, 8)
                    .background(.quaternary, in: RoundedRectangle(cornerRadius: 10))

                Button {
                    NSPasteboard.general.clearContents()
                    NSPasteboard.general.setString(device.userCode, forType: .string)
                } label: {
                    Label("Copy", systemImage: "doc.on.doc")
                }
            }

            Button {
                if let url = URL(string: device.verificationURI) {
                    NSWorkspace.shared.open(url)
                }
            } label: {
                Label("Open \(device.verificationURI)", systemImage: "safari")
            }

            HStack(spacing: 8) {
                ProgressView().controlSize(.small)
                Text(status.isEmpty ? "Waiting for approval…" : status)
                    .font(.callout)
                    .foregroundStyle(.secondary)
            }

            Text("The code expires in about \(device.expiresIn / 60) minutes.")
                .font(.caption2)
                .foregroundStyle(.tertiary)
        }
    }

    // MARK: Token flow

    private var tokenForm: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("Paste a fine-grained or classic token. It is piped to the gh keyring and never stored by ghm.")
                .font(.callout)
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)

            SecureField("github_pat_… or ghp_…", text: $token)
                .textFieldStyle(.roundedBorder)
                .font(.body.monospaced())
                .disabled(busy)

            if let url = URL(string: "https://github.com/settings/tokens/new?scopes=repo,read:org,admin:public_key&description=ghm") {
                Link(destination: url) {
                    Label("Create a token with the right scopes", systemImage: "arrow.up.right.square")
                        .font(.caption)
                }
            }

            Text("Required: repo, read:org, admin:public_key. Without admin:public_key the account is added but the SSH key must be pasted into GitHub manually.")
                .font(.caption2)
                .foregroundStyle(.tertiary)
                .fixedSize(horizontal: false, vertical: true)
        }
    }

    // MARK: Result

    private var resultView: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 8) {
                Image(systemName: result?.ok == true ? "checkmark.circle.fill" : "xmark.octagon.fill")
                    .foregroundStyle(result?.ok == true ? .green : .red)
                Text(result?.ok == true ? "Account ready" : "Could not add the account")
                    .font(.headline)
            }

            ScrollView {
                VStack(alignment: .leading, spacing: 4) {
                    ForEach(Array((result?.lines ?? []).enumerated()), id: \.offset) { _, line in
                        Text(line)
                            .font(.caption.monospaced())
                            .foregroundStyle(line.hasPrefix("warn:") ? .orange : .secondary)
                            .textSelection(.enabled)
                            .frame(maxWidth: .infinity, alignment: .leading)
                    }
                    if let error = result?.error {
                        Text("error: \(error)")
                            .font(.caption.monospaced())
                            .foregroundStyle(.red)
                            .textSelection(.enabled)
                    }
                }
                .padding(10)
            }
            .background(.quaternary.opacity(0.4), in: RoundedRectangle(cornerRadius: 8))
        }
    }

    private func failureView(_ message: String) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            Label("Login failed", systemImage: "xmark.octagon.fill")
                .foregroundStyle(.red)
                .font(.headline)
            Text(message)
                .font(.callout)
                .textSelection(.enabled)
                .fixedSize(horizontal: false, vertical: true)
                .padding(10)
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(.red.opacity(0.08), in: RoundedRectangle(cornerRadius: 8))
        }
    }

    // MARK: Footer

    private var footer: some View {
        HStack {
            if busy && device == nil {
                ProgressView().controlSize(.small)
            }
            Spacer()

            if result != nil {
                Button("Add another") { reset() }
                Button("Done") {
                    dismiss()
                }
                .keyboardShortcut(.defaultAction)
            } else if device != nil {
                Button("Cancel", role: .cancel) { cancelDevice() }
            } else if failure != nil {
                Button("Try again") { reset() }
                Button("Close") { dismiss() }
                    .keyboardShortcut(.defaultAction)
            } else {
                Button("Cancel", role: .cancel) { dismiss() }
                if method == .browser {
                    Button(busy ? "Starting…" : "Start browser login") { startDevice() }
                        .keyboardShortcut(.defaultAction)
                        .disabled(busy)
                } else {
                    Button("Add token") { submitToken() }
                        .keyboardShortcut(.defaultAction)
                        .disabled(busy || token.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
        }
    }

    // MARK: Actions

    private func startDevice() {
        busy = true
        failure = nil
        Task { @MainActor in
            do {
                let code = try await GhmCLI.shared.deviceStart(clientID: clientID)
                if let error = code.error {
                    throw CLIError.failed(error)
                }
                device = code
                status = "Waiting for approval…"
                busy = false
                if let url = URL(string: code.verificationURI) {
                    NSWorkspace.shared.open(url)
                }
                poll(code)
            } catch {
                busy = false
                failure = error.localizedDescription
            }
        }
    }

    private func poll(_ code: DeviceCode) {
        pollTask?.cancel()
        pollTask = Task { @MainActor in
            var interval = max(code.interval, 5)
            let deadline = Date().addingTimeInterval(TimeInterval(code.expiresIn))
            while !Task.isCancelled && Date() < deadline {
                try? await Task.sleep(nanoseconds: UInt64(interval) * 1_000_000_000)
                if Task.isCancelled { return }
                do {
                    let poll = try await GhmCLI.shared.devicePoll(clientID: clientID, deviceCode: code.deviceCode)
                    switch poll.status {
                    case "pending":
                        status = "Waiting for approval…"
                    case "slow_down":
                        interval += 5
                        status = "Waiting for approval…"
                    case "success":
                        status = ""
                        result = poll.result
                        device = nil
                        await model.refresh(silently: true)
                        return
                    default:
                        failure = poll.result.error ?? "GitHub rejected the login."
                        device = nil
                        return
                    }
                } catch {
                    failure = error.localizedDescription
                    device = nil
                    return
                }
            }
            if !Task.isCancelled {
                failure = "The device code expired. Start again."
                device = nil
            }
        }
    }

    private func submitToken() {
        let value = token.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !value.isEmpty else { return }
        busy = true
        failure = nil
        Task { @MainActor in
            do {
                let added = try await GhmCLI.shared.loginToken(value)
                token = ""
                result = added
                await model.refresh(silently: true)
            } catch {
                failure = error.localizedDescription
            }
            busy = false
        }
    }

    private func cancelDevice() {
        pollTask?.cancel()
        pollTask = nil
        device = nil
        status = ""
        busy = false
    }

    private func reset() {
        pollTask?.cancel()
        pollTask = nil
        device = nil
        result = nil
        failure = nil
        token = ""
        busy = false
        status = ""
    }
}
