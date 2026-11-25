import SwiftUI

extension Color {
    init(hex: UInt32, alpha: Double = 1) {
        self.init(
            .sRGB,
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255,
            opacity: alpha
        )
    }

    static let ghmIndigo = Color(hex: 0x5B5CFF)
    static let ghmViolet = Color(hex: 0xA855F7)
    static let ghmTeal = Color(hex: 0x2DD4BF)
    static let ghmMint = Color(hex: 0x34D399)
    static let ghmSky = Color(hex: 0x38BDF8)
}

/// Full-window backdrop. The glass panels refract these colours, which is what
/// makes Liquid Glass read as glass instead of grey blur.
struct LiquidBackground: View {
    @State private var drift = false

    var body: some View {
        ZStack {
            LinearGradient(
                colors: [Color(hex: 0x070A18), Color(hex: 0x101534), Color(hex: 0x081820)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )

            GeometryReader { geo in
                let w = geo.size.width
                let h = geo.size.height
                Circle()
                    .fill(Color.ghmIndigo.opacity(0.55))
                    .frame(width: w * 0.55)
                    .blur(radius: 130)
                    .offset(x: drift ? -w * 0.12 : -w * 0.22, y: drift ? -h * 0.28 : -h * 0.18)

                Circle()
                    .fill(Color.ghmViolet.opacity(0.45))
                    .frame(width: w * 0.5)
                    .blur(radius: 150)
                    .offset(x: drift ? w * 0.42 : w * 0.3, y: drift ? -h * 0.35 : -h * 0.22)

                Circle()
                    .fill(Color.ghmTeal.opacity(0.35))
                    .frame(width: w * 0.6)
                    .blur(radius: 160)
                    .offset(x: drift ? w * 0.1 : w * 0.02, y: drift ? h * 0.42 : h * 0.3)
            }
            .ignoresSafeArea()

            // A faint noise-like sheen so the flat gradient does not band.
            LinearGradient(
                colors: [.white.opacity(0.08), .clear, .white.opacity(0.03)],
                startPoint: .top,
                endPoint: .bottom
            )
            .ignoresSafeArea()
        }
        .ignoresSafeArea()
        .onAppear {
            withAnimation(.easeInOut(duration: 22).repeatForever(autoreverses: true)) {
                drift = true
            }
        }
    }
}

extension View {
    /// Regular Liquid Glass panel, optionally tinted.
    func glassPanel(_ radius: CGFloat = 22, tint: Color? = nil, interactive: Bool = false) -> some View {
        modifier(GlassPanelModifier(radius: radius, tint: tint, interactive: interactive))
    }
}

private struct GlassPanelModifier: ViewModifier {
    let radius: CGFloat
    let tint: Color?
    let interactive: Bool

    func body(content: Content) -> some View {
        content.glassEffect(glass, in: .rect(cornerRadius: radius))
    }

    private var glass: Glass {
        var glass: Glass = .regular
        if let tint {
            glass = glass.tint(tint)
        }
        if interactive {
            glass = glass.interactive()
        }
        return glass
    }
}

/// A quiet glass capsule used for counters and status text.
struct GlassBadge: View {
    let text: String
    var systemImage: String?
    var tint: Color = .secondary

    var body: some View {
        HStack(spacing: 5) {
            if let systemImage {
                Image(systemName: systemImage)
                    .font(.caption2.weight(.semibold))
            }
            Text(text)
                .font(.caption.weight(.medium).monospacedDigit())
        }
        .foregroundStyle(tint)
        .padding(.horizontal, 9)
        .padding(.vertical, 4)
        .glassEffect(.regular, in: .capsule)
    }
}

/// Compact icon-only glass button used in headers.
struct GlassIconButton: View {
    let systemImage: String
    let help: String
    var action: () -> Void

    var body: some View {
        Button(action: action) {
            Image(systemName: systemImage)
                .font(.system(size: 13, weight: .semibold))
                .frame(width: 18, height: 18)
        }
        .buttonStyle(.glass)
        .help(help)
    }
}

struct StatePill: View {
    let repo: Repo

    var body: some View {
        Text(repo.stateLabel)
            .font(.caption2.weight(.semibold))
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .foregroundStyle(foreground)
            .background(background, in: .capsule)
            .overlay(
                Capsule().strokeBorder(.white.opacity(0.12), lineWidth: 0.5)
            )
    }

    private var foreground: Color {
        if repo.dirty > 0 { return .orange }
        if repo.behind > 0 { return .ghmSky }
        if repo.ahead > 0 { return .ghmMint }
        return .secondary
    }

    private var background: Color {
        if repo.dirty > 0 { return .orange.opacity(0.16) }
        if repo.behind > 0 { return .ghmSky.opacity(0.16) }
        if repo.ahead > 0 { return .ghmMint.opacity(0.16) }
        return .white.opacity(0.07)
    }
}

struct Avatar: View {
    let account: Account
    var size: CGFloat = 26

    var body: some View {
        ZStack {
            Circle().fill(
                LinearGradient(
                    colors: tint,
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
            Text(account.initials)
                .font(.system(size: size * 0.36, weight: .bold))
                .foregroundStyle(.white)
        }
        .frame(width: size, height: size)
        .overlay(Circle().strokeBorder(.white.opacity(0.35), lineWidth: 0.8))
        .shadow(color: tint.first?.opacity(0.35) ?? .clear, radius: 6, y: 2)
    }

    private var tint: [Color] {
        let palette: [[Color]] = [
            [.ghmSky, .ghmIndigo],
            [.ghmViolet, .pink],
            [.ghmTeal, .ghmMint],
        ]
        let sum = account.login.unicodeScalars.reduce(0) { $0 + Int($1.value) }
        return palette[sum % palette.count]
    }
}
