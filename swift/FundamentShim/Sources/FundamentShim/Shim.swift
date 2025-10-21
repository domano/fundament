import Foundation
#if canImport(FoundationModels)
import FoundationModels
#endif

@frozen
public struct fundament_error {
    public var code: Int32
    public var message: UnsafePointer<CChar>?
    public init(code: Int32, message: UnsafePointer<CChar>?) {
        self.code = code
        self.message = message
    }
}

@frozen
public struct fundament_buffer {
    public var data: UnsafePointer<CChar>?
    public var length: Int64
    public init(data: UnsafePointer<CChar>?, length: Int64) {
        self.data = data
        self.length = length
    }
}

@frozen
public struct fundament_availability {
    public var state: Int32
    public var reason: Int32
    public init(state: Int32, reason: Int32) {
        self.state = state
        self.reason = reason
    }
}

public typealias fundament_stream_cb = @convention(c) (UnsafePointer<CChar>?, Bool, UnsafeMutableRawPointer?) -> Void

#if canImport(FoundationModels)
@available(macOS 26.0, *)
private final class SessionBox: @unchecked Sendable {
    let session: LanguageModelSession
    init(session: LanguageModelSession) {
        self.session = session
    }
}
#endif

// MARK: - Helpers

private func duplicateCString(_ string: String) -> UnsafePointer<CChar>? {
    guard let duplicated = string.withCString({ strdup($0) }) else {
        return nil
    }
    return UnsafePointer(duplicated)
}

private func setError(_ error: Error, into target: UnsafeMutablePointer<fundament_error>?) {
    guard let target else { return }
    let nsError = error as NSError
    let messagePointer = duplicateCString("\(nsError.domain)(\(nsError.code)): \(nsError.localizedDescription)")
    target.pointee = fundament_error(code: Int32(nsError.code), message: messagePointer)
}

private func setUnavailableError(into target: UnsafeMutablePointer<fundament_error>?, message: String) {
    guard let target else { return }
    target.pointee = fundament_error(code: -1, message: duplicateCString(message))
}

private func wrapBuffer(from string: String) -> fundament_buffer {
    fundament_buffer(data: duplicateCString(string), length: Int64(string.utf8.count))
}

private func parseString(_ pointer: UnsafePointer<CChar>?) -> String {
    guard let pointer else { return "" }
    return String(cString: pointer)
}

private func bindErrorPointer(_ raw: UnsafeMutableRawPointer?) -> UnsafeMutablePointer<fundament_error>? {
    raw?.assumingMemoryBound(to: fundament_error.self)
}

private func bindAvailabilityPointer(_ raw: UnsafeMutableRawPointer?) -> UnsafeMutablePointer<fundament_availability>? {
    raw?.assumingMemoryBound(to: fundament_availability.self)
}

private func bindBufferPointer(_ raw: UnsafeMutableRawPointer?) -> UnsafeMutablePointer<fundament_buffer>? {
    raw?.assumingMemoryBound(to: fundament_buffer.self)
}

private func clearBuffer(_ pointer: UnsafeMutablePointer<fundament_buffer>?) {
    guard let pointer else { return }
    if let existing = pointer.pointee.data {
        UnsafeMutablePointer(mutating: existing).deallocate()
    }
    pointer.pointee = fundament_buffer(data: nil, length: 0)
}

private func storeString(_ string: String, in pointer: UnsafeMutablePointer<fundament_buffer>?) {
    guard let pointer else { return }
    clearBuffer(pointer)
    pointer.pointee = wrapBuffer(from: string)
}

#if canImport(FoundationModels)
@available(macOS 26.0, *)
private func makeGenerationOptions(from _: String) -> GenerationOptions {
    GenerationOptions()
}

@available(macOS 26.0, *)
private func withSessionBox(_ ref: UnsafeMutableRawPointer?) -> SessionBox? {
    guard let ref else { return nil }
    return Unmanaged<SessionBox>.fromOpaque(ref).takeUnretainedValue()
}

@preconcurrency
@available(macOS 26.0, *)
private func performSync<T>(_ operation: @escaping @Sendable () async throws -> T) throws -> T {
    let semaphore = DispatchSemaphore(value: 0)
    var result: Result<T, Error> = .failure(NSError(domain: "dev.fundament.shim", code: -1, userInfo: [NSLocalizedDescriptionKey: "Unknown error"]))
    Task {
        do {
            let value = try await operation()
            result = .success(value)
        } catch {
            result = .failure(error)
        }
        semaphore.signal()
    }
    semaphore.wait()
    return try result.get()
}

@available(macOS 26.0, *)
private func checkAvailability() -> fundament_availability {
    let model = SystemLanguageModel.default
    switch model.availability {
    case .available:
        return fundament_availability(state: 1, reason: 0)
    case .unavailable(let reason):
        let reasonValue: Int32
        switch reason {
        case .deviceNotEligible:
            reasonValue = 1
        case .appleIntelligenceNotEnabled:
            reasonValue = 2
        case .modelNotReady:
            reasonValue = 3
        @unknown default:
            reasonValue = -1
        }
        return fundament_availability(state: 0, reason: reasonValue)
    }
}
#endif

// MARK: - Exported C functions

@_cdecl("fundament_session_create")
public func fundament_session_create(_ instructions: UnsafePointer<CChar>?, _ outError: UnsafeMutableRawPointer?) -> UnsafeMutableRawPointer? {
#if canImport(FoundationModels)
    let errorPtr = bindErrorPointer(outError)
    guard #available(macOS 26.0, *) else {
        setUnavailableError(into: errorPtr, message: "SystemLanguageModel requires macOS 26.0 or newer.")
        return nil
    }
    let instructionsString = parseString(instructions)
    let session = LanguageModelSession(instructions: instructionsString)
    let box = SessionBox(session: session)
    return Unmanaged.passRetained(box).toOpaque()
#else
    setUnavailableError(into: bindErrorPointer(outError), message: "FoundationModels framework is unavailable on this platform.")
    return nil
#endif
}

@_cdecl("fundament_session_destroy")
public func fundament_session_destroy(_ ref: UnsafeMutableRawPointer?) {
#if canImport(FoundationModels)
    guard let ref, #available(macOS 26.0, *) else { return }
    Unmanaged<SessionBox>.fromOpaque(ref).release()
#endif
}

@_cdecl("fundament_session_check_availability")
public func fundament_session_check_availability(_ outAvailability: UnsafeMutableRawPointer?, _ outError: UnsafeMutableRawPointer?) -> Bool {
#if canImport(FoundationModels)
    let availabilityPtr = bindAvailabilityPointer(outAvailability)
    let errorPtr = bindErrorPointer(outError)
    guard #available(macOS 26.0, *) else {
        setUnavailableError(into: errorPtr, message: "SystemLanguageModel requires macOS 26.0 or newer.")
        return false
    }
    availabilityPtr?.pointee = checkAvailability()
    return true
#else
    setUnavailableError(into: bindErrorPointer(outError), message: "FoundationModels framework is unavailable on this platform.")
    return false
#endif
}

@_cdecl("fundament_session_respond")
public func fundament_session_respond(_ ref: UnsafeMutableRawPointer?, _ prompt: UnsafePointer<CChar>?, _ optionsJSON: UnsafePointer<CChar>?, _ outBuffer: UnsafeMutableRawPointer?, _ outError: UnsafeMutableRawPointer?) -> Bool {
#if canImport(FoundationModels)
    let bufferPtr = bindBufferPointer(outBuffer)
    let errorPtr = bindErrorPointer(outError)
    guard #available(macOS 26.0, *) else {
        setUnavailableError(into: errorPtr, message: "SystemLanguageModel requires macOS 26.0 or newer.")
        clearBuffer(bufferPtr)
        return false
    }
    guard let box = withSessionBox(ref) else {
        setUnavailableError(into: errorPtr, message: "Invalid session handle.")
        clearBuffer(bufferPtr)
        return false
    }
    do {
        let promptString = parseString(prompt)
        let options = makeGenerationOptions(from: parseString(optionsJSON))
        let response = try performSync {
            try await box.session.respond(to: promptString, options: options)
        }
        storeString(response.content, in: bufferPtr)
        return true
    } catch {
        setError(error, into: errorPtr)
        clearBuffer(bufferPtr)
        return false
    }
#else
    setUnavailableError(into: bindErrorPointer(outError), message: "FoundationModels framework is unavailable on this platform.")
    clearBuffer(bindBufferPointer(outBuffer))
    return false
#endif
}

@_cdecl("fundament_session_respond_structured")
public func fundament_session_respond_structured(_ ref: UnsafeMutableRawPointer?, _ prompt: UnsafePointer<CChar>?, _ schemaJSON: UnsafePointer<CChar>?, _ optionsJSON: UnsafePointer<CChar>?, _ outBuffer: UnsafeMutableRawPointer?, _ outError: UnsafeMutableRawPointer?) -> Bool {
#if canImport(FoundationModels)
    let bufferPtr = bindBufferPointer(outBuffer)
    let errorPtr = bindErrorPointer(outError)
    guard #available(macOS 26.0, *) else {
        setUnavailableError(into: errorPtr, message: "SystemLanguageModel requires macOS 26.0 or newer.")
        clearBuffer(bufferPtr)
        return false
    }
    guard let box = withSessionBox(ref) else {
        setUnavailableError(into: errorPtr, message: "Invalid session handle.")
        clearBuffer(bufferPtr)
        return false
    }
    do {
        let promptString = parseString(prompt)
        let schemaString = parseString(schemaJSON)
        let options = makeGenerationOptions(from: parseString(optionsJSON))
        let response = try performSync {
            let schema = try decodeSchema(from: schemaString)
            return try await box.session.respond(to: promptString, schema: schema, includeSchemaInPrompt: true, options: options)
        }
        let raw = response.rawContent
        let json = raw.jsonString
        storeString(json, in: bufferPtr)
        return true
    } catch {
        setError(error, into: errorPtr)
        clearBuffer(bufferPtr)
        return false
    }
#else
    setUnavailableError(into: bindErrorPointer(outError), message: "FoundationModels framework is unavailable on this platform.")
    clearBuffer(bindBufferPointer(outBuffer))
    return false
#endif
}

@_cdecl("fundament_session_stream")
public func fundament_session_stream(_ ref: UnsafeMutableRawPointer?, _ prompt: UnsafePointer<CChar>?, _ optionsJSON: UnsafePointer<CChar>?, _ callback: fundament_stream_cb?, _ userData: UnsafeMutableRawPointer?, _ outError: UnsafeMutableRawPointer?) -> Bool {
#if canImport(FoundationModels)
    let errorPtr = bindErrorPointer(outError)
    guard #available(macOS 26.0, *) else {
        setUnavailableError(into: errorPtr, message: "SystemLanguageModel requires macOS 26.0 or newer.")
        return false
    }
    guard let box = withSessionBox(ref) else {
        setUnavailableError(into: errorPtr, message: "Invalid session handle.")
        return false
    }
    guard let callback else {
        setUnavailableError(into: errorPtr, message: "Callback is required.")
        return false
    }
    let promptString = parseString(prompt)
    let options = makeGenerationOptions(from: parseString(optionsJSON))
    let streamContext = StreamContext(userData: userData)
    do {
        _ = try performSync {
            let stream = box.session.streamResponse(to: promptString, options: options)
            let response = try await stream.collect()
            try await callStreamingCallback(with: response.content, callback: callback, userData: streamContext.userData)
            return true
        }
        return true
    } catch {
        setError(error, into: errorPtr)
        return false
    }
#else
    setUnavailableError(into: bindErrorPointer(outError), message: "FoundationModels framework is unavailable on this platform.")
    return false
#endif
}

@_cdecl("fundament_buffer_free")
public func fundament_buffer_free(_ raw: UnsafeMutableRawPointer?) {
#if canImport(FoundationModels)
    clearBuffer(bindBufferPointer(raw))
#else
    clearBuffer(bindBufferPointer(raw))
#endif
}

@_cdecl("fundament_error_free")
public func fundament_error_free(_ raw: UnsafeMutableRawPointer?) {
    guard let pointer = bindErrorPointer(raw) else { return }
    if let message = pointer.pointee.message {
        UnsafeMutablePointer(mutating: message).deallocate()
    }
    pointer.pointee = fundament_error(code: 0, message: nil)
}

#if canImport(FoundationModels)
@available(macOS 26.0, *)
private final class SchemaNode: Decodable {
    final class Property: Decodable {
        let name: String
        let schema: SchemaNode
    }

    let name: String?
    let description: String?
    let type: String?
    let properties: [Property]?
    let items: SchemaNode?
    let minimumElements: Int?
    let maximumElements: Int?
    let anyOf: [String]?
}

@available(macOS 26.0, *)
private func buildDynamicSchema(from node: SchemaNode) throws -> DynamicGenerationSchema {
    if let properties = node.properties, !properties.isEmpty {
        let dynamicProperties = try properties.map {
            DynamicGenerationSchema.Property(name: $0.name, schema: try buildDynamicSchema(from: $0.schema))
        }
        return DynamicGenerationSchema(name: node.name ?? "Object", description: node.description, properties: dynamicProperties)
    }

    if node.type == "array" {
        guard let element = node.items else {
            throw NSError(domain: "dev.fundament.shim", code: -6, userInfo: [NSLocalizedDescriptionKey: "Array schema requires 'items'"])
        }
        let dynamicElement = try buildDynamicSchema(from: element)
        return DynamicGenerationSchema(arrayOf: dynamicElement, minimumElements: node.minimumElements, maximumElements: node.maximumElements)
    }

    switch node.type {
    case "string", nil:
        if let anyOf = node.anyOf {
            return DynamicGenerationSchema(name: node.name ?? "String", description: node.description, anyOf: anyOf)
        }
        return DynamicGenerationSchema(type: String.self, guides: [])
    case "integer":
        return DynamicGenerationSchema(type: Int.self, guides: [])
    case "boolean":
        return DynamicGenerationSchema(type: Bool.self, guides: [])
    default:
        throw NSError(domain: "dev.fundament.shim", code: -7, userInfo: [NSLocalizedDescriptionKey: "Unsupported schema type '\(node.type ?? "unknown")'"])
    }
}

@available(macOS 26.0, *)
private struct StreamContext: @unchecked Sendable {
    let userData: UnsafeMutableRawPointer?
}

@available(macOS 26.0, *)
private func decodeSchema(from json: String) throws -> GenerationSchema {
    guard !json.isEmpty else {
        throw NSError(domain: "dev.fundament.shim", code: -2, userInfo: [NSLocalizedDescriptionKey: "Schema JSON required for structured generation"])
    }
    let data = Data(json.utf8)
    let decoder = JSONDecoder()
    let node = try decoder.decode(SchemaNode.self, from: data)
    let root = try buildDynamicSchema(from: node)
    return try GenerationSchema(root: root, dependencies: [])
}

@available(macOS 26.0, *)
private func callStreamingCallback(with text: String, callback: fundament_stream_cb, userData: UnsafeMutableRawPointer?) async throws {
    let components = text.split(separator: " ").map(String.init)
    if components.isEmpty {
        let pointer = duplicateCString(text)
        callback(pointer, true, userData)
        if let pointer {
            UnsafeMutablePointer(mutating: pointer).deallocate()
        }
        return
    }

    for (index, chunk) in components.enumerated() {
        let pointer = duplicateCString(chunk)
        callback(pointer, index == components.count - 1, userData)
        if let pointer {
            UnsafeMutablePointer(mutating: pointer).deallocate()
        }
    }
}

#endif
