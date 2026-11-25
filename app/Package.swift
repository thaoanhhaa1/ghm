// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "GhmApp",
    platforms: [.macOS("26.0")],
    products: [
        .executable(name: "GhmApp", targets: ["GhmApp"])
    ],
    dependencies: [
        .package(url: "https://github.com/sparkle-project/Sparkle", from: "2.6.0")
    ],
    targets: [
        .executableTarget(
            name: "GhmApp",
            dependencies: [
                .product(name: "Sparkle", package: "Sparkle")
            ],
            path: "Sources/GhmApp"
        )
    ]
)
