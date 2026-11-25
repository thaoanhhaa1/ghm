// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "GhmApp",
    platforms: [.macOS(.v13)],
    products: [
        .executable(name: "GhmApp", targets: ["GhmApp"])
    ],
    targets: [
        .executableTarget(
            name: "GhmApp",
            path: "Sources/GhmApp"
        )
    ]
)
