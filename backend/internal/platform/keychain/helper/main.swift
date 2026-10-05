import Foundation
import Security

struct Request: Decodable {
    let operation: String
    let account: String
    let key: String?
}

struct Response: Encodable {
    var key: String? = nil
    var error: String? = nil
}

func perform(_ request: Request) -> Response {
    guard !request.account.isEmpty else { return Response(error: "Invalid credential account") }
    let query: [String: Any] = [
        kSecClass as String: kSecClassGenericPassword,
        kSecAttrService as String: "com.danncd.arx.deepseek",
        kSecAttrAccount as String: request.account
    ]
    var status: OSStatus
    switch request.operation {
    case "load":
        var lookup = query
        lookup[kSecReturnData as String] = true
        lookup[kSecMatchLimit as String] = kSecMatchLimitOne
        var result: CFTypeRef?
        status = SecItemCopyMatching(lookup as CFDictionary, &result)
        if status == errSecItemNotFound { return Response() }
        if status == errSecSuccess, let data = result as? Data, let key = String(data: data, encoding: .utf8) {
            return Response(key: key)
        }
    case "save":
        guard let key = request.key, !key.isEmpty else { return Response(error: "An API key is required") }
        let value = [kSecValueData as String: Data(key.utf8)]
        status = SecItemUpdate(query as CFDictionary, value as CFDictionary)
        if status == errSecItemNotFound {
            var item = query
            item[kSecValueData as String] = Data(key.utf8)
            item[kSecAttrLabel as String] = "Arx — DeepSeek API key"
            status = SecItemAdd(item as CFDictionary, nil)
        }
    case "delete":
        status = SecItemDelete(query as CFDictionary)
        if status == errSecItemNotFound { return Response() }
    default:
        return Response(error: "Unknown credential operation")
    }
    return status == errSecSuccess ? Response() : Response(error: "macOS Keychain is unavailable or access was denied")
}

let response: Response
do {
    let data = FileHandle.standardInput.readDataToEndOfFile()
    guard data.count <= 16384 else { throw NSError(domain: "request", code: 1) }
    let request = try JSONDecoder().decode(Request.self, from: data)
    response = perform(request)
} catch {
    response = Response(error: "Invalid credential request")
}
if let encoded = try? JSONEncoder().encode(response) {
    FileHandle.standardOutput.write(encoded)
}
