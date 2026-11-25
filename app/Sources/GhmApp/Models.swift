import Foundation

struct Account: Identifiable, Decodable, Hashable {
    let login: String
    let name: String
    let email: String
    let dir: String
    let key: String
    let hostAlias: String
    let keyOK: Bool
    let tokenOK: Bool
    let repos: Int

    var id: String { login }

    enum CodingKeys: String, CodingKey {
        case login, name, email, dir, key, repos
        case hostAlias = "host_alias"
        case keyOK = "key_ok"
        case tokenOK = "token_ok"
    }

    var initials: String {
        let parts = name.split(separator: " ").prefix(2)
        let letters = parts.compactMap { $0.first }.map(String.init).joined()
        return letters.isEmpty ? String(login.prefix(2)).uppercased() : letters.uppercased()
    }

    var tildeDir: String { tilde(dir) }
}

struct Repo: Identifiable, Decodable, Hashable {
    let path: String
    let slug: String
    let login: String
    let branch: String
    let origin: String
    let ahead: Int
    let behind: Int
    let dirty: Int
    let isGitHub: Bool

    var id: String { path }

    enum CodingKeys: String, CodingKey {
        case path, slug, login, branch, origin, ahead, behind, dirty
        case isGitHub = "is_github"
    }

    var displayName: String {
        if !slug.isEmpty { return slug }
        return tilde(path)
    }

    var shortName: String {
        displayName.split(separator: "/").last.map(String.init) ?? displayName
    }

    var stateLabel: String {
        if dirty > 0 { return "\(dirty) dirty" }
        if behind > 0 { return "behind" }
        if ahead > 0 { return "ahead" }
        return "clean"
    }

    var tildePath: String { tilde(path) }

    var githubURL: URL? {
        slug.isEmpty ? nil : URL(string: "https://github.com/\(slug)")
    }
}

struct StatePayload: Decodable {
    let active: String
    let root: String
    let accounts: [Account]
    let repos: [Repo]
}

struct DoctorCheck: Decodable, Identifiable {
    let name: String
    let level: String
    let detail: String
    var id: String { name }

    var symbol: String {
        switch level {
        case "ok": return "checkmark.circle.fill"
        case "warn": return "exclamationmark.triangle.fill"
        default: return "xmark.octagon.fill"
        }
    }
}

struct DoctorPayload: Decodable {
    let checks: [DoctorCheck]
    let ok: Int
    let warn: Int
    let fail: Int
}

struct APIResult: Decodable {
    let ok: Bool?
    let logins: [String]?
    let lines: [String]?
    let error: String?
    let output: String?
}

struct DeviceCode: Decodable {
    let deviceCode: String
    let userCode: String
    let verificationURI: String
    let expiresIn: Int
    let interval: Int
    let error: String?

    enum CodingKeys: String, CodingKey {
        case deviceCode = "device_code"
        case userCode = "user_code"
        case verificationURI = "verification_uri"
        case expiresIn = "expires_in"
        case interval, error
    }
}

struct DevicePoll: Decodable {
    let status: String
    let result: APIResult
}

func tilde(_ path: String) -> String {
    let home = NSHomeDirectory()
    if path == home { return "~" }
    if path.hasPrefix(home + "/") { return "~" + path.dropFirst(home.count) }
    return path
}
