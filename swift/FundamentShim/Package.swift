// swift-tools-version: 5.10
import PackageDescription

let package = Package(
    name: "FundamentShim",
    platforms: [
        .macOS(.v14)
    ],
    products: [
        .library(
            name: "FundamentShim",
            type: .dynamic,
            targets: ["FundamentShim"]
        )
    ],
    targets: [
        .target(
            name: "FundamentShim",
            dependencies: [],
            path: "Sources",
            swiftSettings: [
                .enableExperimentalFeature("StrictConcurrency"),
                .enableUpcomingFeature("ConciseMagicFile"),
                .define("FUNDAMENT_SHIM")
            ],
            linkerSettings: [
                .linkedFramework("Foundation"),
                .linkedFramework("FoundationModels")
            ]
        )
    ]
)
