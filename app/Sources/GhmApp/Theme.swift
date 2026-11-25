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

    // Graphite glass palette: one restrained steel accent, everything else
    // desaturated so the window reads as a macOS tool rather than artwork.
    static let ghmAccent = Color(hex: 0x9DB3CC)
    static let ghmPositive = Color(hex: 0x8FB39B)
    static let ghmWarning = Color(hex: 0xC4A472)
    static let ghmInfo = Color(hex: 0x93A9C4)
    static let ghmDanger = Color(hex: 0xC49090)
}

/// Full-window backdrop. Kept almost neutral: the glass panels need something
/// to refract, but the colour should never compete with the repository list.
struct LiquidBackground: View {
    @State private var drift = false

    var body: some View {
        ZStack {
            LinearGradient(
                colors: [Color(hex: 0x0B0D11), Color(hex: 0x11141A), Color(hex: 0x0C0F14)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )

            GeometryReader { geo in
                let w = geo.size.width
                let h = geo.size.height
                Circle()
                    .fill(Color(hex: 0x5C708C).opacity(0.20))
                    .frame(width: w * 0.6)
                    .blur(radius: 190)
                    .offset(x: drift ? -w * 0.18 : -w * 0.24, y: drift ? -h * 0.26 : -h * 0.2)

                Circle()
                    .fill(Color(hex: 0x7A7468).opacity(0.12))
                    .frame(width: w * 0.5)
                    .blur(radius: 200)
                    .offset(x: drift ? w * 0.36 : w * 0.28, y: drift ? h * 0.3 : h * 0.22)
            }
            .ignoresSafeArea()

            // Vignette: pulls the eye to the centre and hides any gradient banding.
            RadialGradient(
                colors: [.clear, .black.opacity(0.28)],
                center: .center,
                startRadius: 160,
                endRadius: 900
            )
            .ignoresSafeArea()
        }
        .ignoresSafeArea()
        .onAppear {
            withAnimation(.easeInOut(duration: 30).repeatForever(autoreverses: true)) {
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
            .font(.caption2.weight(.medium))
            .padding(.horizontal, 8)
            .padding(.vertical, 3)
            .foregroundStyle(foreground)
            .background(background, in: .capsule)
            .overlay(
                Capsule().strokeBorder(.white.opacity(0.08), lineWidth: 0.5)
            )
    }

    private var foreground: Color {
        if repo.dirty > 0 { return .ghmWarning }
        if repo.behind > 0 { return .ghmInfo }
        if repo.ahead > 0 { return .ghmPositive }
        return .secondary
    }

    private var background: Color {
        if repo.dirty > 0 { return .ghmWarning.opacity(0.12) }
        if repo.behind > 0 { return .ghmInfo.opacity(0.12) }
        if repo.ahead > 0 { return .ghmPositive.opacity(0.12) }
        return .white.opacity(0.05)
    }
}

struct Avatar: View {
    let account: Account
    var size: CGFloat = 26

    var body: some View {
        ZStack {
            Circle().fill(
                LinearGradient(
                    colors: [tint.opacity(0.85), tint.opacity(0.45)],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
            Text(account.initials)
                .font(.system(size: size * 0.36, weight: .semibold))
                .foregroundStyle(.white.opacity(0.92))
        }
        .frame(width: size, height: size)
        .overlay(Circle().strokeBorder(.white.opacity(0.18), lineWidth: 0.8))
    }

    /// Muted stone tones, picked per login so accounts stay distinguishable
    /// without turning the sidebar into a colour chart.
    private var tint: Color {
        let palette: [Color] = [
            Color(hex: 0x6B7C93),
            Color(hex: 0x7C7468),
            Color(hex: 0x6E8076),
            Color(hex: 0x7A6E7E),
        ]
        let sum = account.login.unicodeScalars.reduce(0) { $0 + Int($1.value) }
        return palette[sum % palette.count]
    }
}
