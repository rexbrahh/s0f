// swift-tools-version: 5.10
import PackageDescription

let package = Package(
    name: "S0FApp",
    platforms: [
        .macOS(.v13)
    ],
    products: [
        .executable(name: "S0FApp", targets: ["S0FApp"])
    ],
    dependencies: [],
    targets: [
        .executableTarget(
            name: "S0FApp",
            path: "S0FApp",
            resources: [.process("Assets.xcassets")] // placeholder
        )
    ]
)
