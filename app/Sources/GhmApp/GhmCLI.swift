import Foundation

enum CLIError: LocalizedError {
    case binaryNotFound
    case failed(String)

    var errorDescription: String? {
        switch self {
        case .binaryNotFound:
            return "ghm binary not found. Install it with `go install github.com/thaoanhhaa1/ghm/cmd/ghm@latest` or `make build`."
        case .failed(let message):
            return message
        }
    }
}

/// Thin bridge to the `ghm` CLI. All state and mutations go through its JSON API,
/// so the app never touches keys or tokens directly.
struct GhmCLI {
    static let shared = GhmCLI()

    let binary: URL?

    init() {
        var candidates: [String] = []
        if let bundled = Bundle.main.url(forAuxiliaryExecutable: "ghm") {
            candidates.append(bundled.path)
        }
        if let resource = Bundle.main.resourceURL?.appendingPathComponent("ghm").path {
            candidates.append(resource)
        }
        candidates.append(NSHomeDirectory() + "/.local/bin/ghm")
        if let path = ProcessInfo.processInfo.environment["PATH"] {
            for dir in path.split(separator: ":") {
                candidates.append(String(dir) + "/ghm")
            }
        }
        candidates.append(contentsOf: ["/opt/homebrew/bin/ghm", "/usr/local/bin/ghm", "/usr/bin/ghm"])

        binary = candidates
            .first { FileManager.default.isExecutableFile(atPath: $0) }
            .map(URL.init(fileURLWithPath:))
    }

    var binaryPath: String { binary?.path ?? "not found" }

    func run(_ arguments: [String], stdin: Data? = nil) async throws -> Data {
        guard let binary else { throw CLIError.binaryNotFound }
        return try await withCheckedThrowingContinuation { continuation in
            DispatchQueue.global(qos: .userInitiated).async {
                let process = Process()
                process.executableURL = binary
                process.arguments = arguments

                var environment = ProcessInfo.processInfo.environment
                environment["PATH"] = "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:" + NSHomeDirectory() + "/.local/bin"
                // An inherited token would silently override the folder account.
                environment.removeValue(forKey: "GH_TOKEN")
                environment.removeValue(forKey: "GHM_SHIM")
                process.environment = environment

                let stdout = Pipe()
                let stderr = Pipe()
                let input = stdin == nil ? nil : Pipe()
                process.standardOutput = stdout
                process.standardError = stderr
                if let input { process.standardInput = input }

                do {
                    try process.run()
                } catch {
                    continuation.resume(throwing: error)
                    return
                }
                if let stdin, let input {
                    input.fileHandleForWriting.write(stdin)
                    try? input.fileHandleForWriting.close()
                }

                let outData = stdout.fileHandleForReading.readDataToEndOfFile()
                let errData = stderr.fileHandleForReading.readDataToEndOfFile()
                process.waitUntilExit()

                if process.terminationStatus != 0 && outData.isEmpty {
                    let message = String(data: errData, encoding: .utf8) ?? "ghm exited with \(process.terminationStatus)"
                    continuation.resume(throwing: CLIError.failed(message.trimmingCharacters(in: .whitespacesAndNewlines)))
                    return
                }
                continuation.resume(returning: outData)
            }
        }
    }

    func state() async throws -> StatePayload {
        try decode(try await run(["api", "state"]))
    }

    func doctor(fix: Bool = false) async throws -> DoctorPayload {
        var arguments = ["api", "doctor"]
        if fix { arguments.append("--fix") }
        return try decode(try await run(arguments))
    }

    func add(login: String) async throws -> APIResult {
        try decode(try await run(["api", "add", login]))
    }

    func loginToken(_ token: String) async throws -> APIResult {
        try decode(try await run(["api", "login-token"], stdin: Data((token + "\n").utf8)))
    }

    func deviceStart(clientID: String) async throws -> DeviceCode {
        try decode(try await run(["api", "device-start", "--client-id", clientID]))
    }

    func devicePoll(clientID: String, deviceCode: String) async throws -> DevicePoll {
        try decode(try await run([
            "api", "device-poll",
            "--client-id", clientID,
            "--device-code", deviceCode,
        ]))
    }

    func git(path: String, action: String) async throws -> APIResult {
        try decode(try await run(["api", "git", "--path", path, "--action", action]))
    }

    private func decode<T: Decodable>(_ data: Data) throws -> T {
        do {
            return try JSONDecoder().decode(T.self, from: data)
        } catch {
            let raw = String(data: data, encoding: .utf8) ?? "<binary>"
            throw CLIError.failed("Cannot read ghm output: \(error.localizedDescription)\n\(raw)")
        }
    }
}
