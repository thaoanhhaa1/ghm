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
                    .font(.title2)
                    .foregroundStyle(.tint)
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
                .background(.red.opacity(0.08), in: RoundedRectangle(cornerRadius: 8))
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
                .background(.quaternary.opacity(0.35), in: RoundedRectangle(cornerRadius: 8))
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
                    .disabled(busy)
                Button("Fix issues") { run(fix: true) }
                    .disabled(busy)
                Button("Close") { dismiss() }
                    .keyboardShortcut(.defaultAction)
            }
        }
        .padding(20)
        .frame(width: 650, height: 560)
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
