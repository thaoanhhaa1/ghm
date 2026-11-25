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
                    .font(.system(size: 19, weight: .semibold))
                    .foregroundStyle(.white)
                    .frame(width: 44, height: 44)
                    .background(
                        LinearGradient(colors: [.iosTeal, .iosBlue], startPoint: .topLeading, endPoint: .bottomTrailing),
                        in: .rect(cornerRadius: 12)
                    )
                    .overlay(
                        RoundedRectangle(cornerRadius: 12).strokeBorder(.white.opacity(0.30), lineWidth: 0.7)
                    )
                VStack(alignment: .leading, spacing: 2) {
                    Text("Health checks")
                        .font(.system(size: 22, weight: .semibold, design: .rounded))
                    Text("ghm doctor")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                if let payload {
                    GlassBadge(text: "\(payload.ok) ok", systemImage: "checkmark.circle.fill", tint: .ghmPositive)
                    GlassBadge(text: "\(payload.warn) warn", systemImage: "exclamationmark.triangle.fill",
                               tint: payload.warn > 0 ? .ghmWarning : .secondary)
                    GlassBadge(text: "\(payload.fail) fail", systemImage: "xmark.octagon.fill",
                               tint: payload.fail > 0 ? .ghmDanger : .secondary)
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
                .padding(12)
                .frame(maxWidth: .infinity, alignment: .leading)
                .glassEffect(.regular.tint(.iosPink.opacity(0.22)), in: .rect(cornerRadius: 18))
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
                .glassEffect(.regular, in: .rect(cornerRadius: 18))
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
                    .buttonBorderShape(.capsule)
                    .disabled(busy)
                Button("Fix issues") { run(fix: true) }
                    .buttonStyle(.glassProminent)
                    .buttonBorderShape(.capsule)
                    .tint(.iosTeal)
                    .disabled(busy)
                Button("Close") { dismiss() }
                    .buttonStyle(.glass)
                    .buttonBorderShape(.capsule)
                    .keyboardShortcut(.defaultAction)
            }
        }
        .padding(22)
        .frame(width: 680, height: 600)
        .background(SheetBackdrop())
        .task { await runAsync(fix: false) }
    }

    private func color(for level: String) -> Color {
        switch level {
        case "ok": return .ghmPositive
        case "warn": return .ghmWarning
        default: return .ghmDanger
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
