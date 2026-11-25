import SwiftUI

struct DoctorView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var payload: DoctorPayload?
    @State private var busy = false
    @State private var failure: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack(spacing: 10) {
                Image(systemName: "stethoscope")
                    .font(.system(size: 18, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(width: 40, height: 40)
                    .glassPanel(14, tint: .ghmTeal)
                VStack(alignment: .leading, spacing: 2) {
                    Text("Health checks")
                        .font(.title2.bold())
                    Text("ghm doctor")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                if let payload {
                    Text("\(payload.ok) ok")
                        .foregroundStyle(.green)
                    Text("\(payload.warn) warn")
                        .foregroundStyle(payload.warn > 0 ? .orange : .secondary)
                    Text("\(payload.fail) fail")
                        .foregroundStyle(payload.fail > 0 ? .red : .secondary)
                }
            }
            .font(.callout.weight(.medium))

            if let failure {
                VStack(alignment: .leading, spacing: 6) {
                    Label("Could not run ghm doctor", systemImage: "xmark.octagon.fill")
                        .foregroundStyle(.red)
                    Text(failure)
                        .font(.callout)
                        .textSelection(.enabled)
                }
                .padding(10)
                .frame(maxWidth: .infinity, alignment: .leading)
                .glassPanel(14, tint: .red)
            } else if let payload {
                ScrollView {
                    LazyVStack(alignment: .leading, spacing: 0) {
                        ForEach(payload.checks) { check in
                            HStack(alignment: .top, spacing: 10) {
                                Image(systemName: check.symbol)
                                    .foregroundStyle(color(for: check.level))
                                    .frame(width: 16)
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(check.name)
                                        .font(.body.monospaced())
                                    Text(check.detail)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                        .textSelection(.enabled)
                                        .fixedSize(horizontal: false, vertical: true)
                                }
                                Spacer(minLength: 0)
                            }
                            .padding(.vertical, 7)
                            .padding(.horizontal, 10)
                            Divider()
                        }
                    }
                }
                .glassPanel(16)
            } else {
                HStack(spacing: 8) {
                    ProgressView().controlSize(.small)
                    Text("Running checks…").foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            }

            HStack(spacing: 10) {
                if busy {
                    ProgressView().controlSize(.small)
                }
                Spacer()
                Button("Run again") { run(fix: false) }
                    .buttonStyle(.glass)
                    .disabled(busy)
                Button("Fix issues") { run(fix: true) }
                    .buttonStyle(.glassProminent)
                    .tint(.ghmTeal)
                    .disabled(busy)
                Button("Close") { dismiss() }
                    .buttonStyle(.glass)
                    .keyboardShortcut(.defaultAction)
            }
        }
        .padding(22)
        .frame(width: 680, height: 600)
        .background(
            ZStack {
                Color(hex: 0x0A0E20)
                LinearGradient(
                    colors: [Color.ghmIndigo.opacity(0.4), Color.ghmTeal.opacity(0.18)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            }
            .ignoresSafeArea()
        )
        .task { await runAsync(fix: false) }
    }

    private func color(for level: String) -> Color {
        switch level {
        case "ok": return .green
        case "warn": return .orange
        default: return .red
        }
    }

    private func run(fix: Bool) {
        Task { await runAsync(fix: fix) }
    }

    private func runAsync(fix: Bool) async {
        busy = true
        failure = nil
        defer { busy = false }
        do {
            payload = try await GhmCLI.shared.doctor(fix: fix)
        } catch {
            failure = error.localizedDescription
        }
    }
}
