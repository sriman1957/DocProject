import 'dart:convert';

import 'package:http/http.dart' as http;

import '../config/api_config.dart';

class ApiException implements Exception {
  const ApiException({required this.statusCode, required this.message});

  final int statusCode;
  final String message;

  @override
  String toString() => 'ApiException($statusCode): $message';
}

class ApiClient {
  ApiClient({String? baseUrl, http.Client? httpClient})
    : _baseUrl = (baseUrl ?? ApiConfig.baseUrl).replaceFirst(
        RegExp(r'/+$'),
        '',
      ),
      _httpClient = httpClient ?? http.Client();

  final String _baseUrl;
  final http.Client _httpClient;

  Uri _buildUri(String path) {
    final normalizedPath = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$_baseUrl$normalizedPath');
  }

  Map<String, String> _headers(String? accessToken) {
    return {
      'Accept': 'application/json',
      if (accessToken != null && accessToken.isNotEmpty)
        'Authorization': 'Bearer $accessToken',
    };
  }

  Future<dynamic> get(String path, {String? accessToken}) {
    return _send('GET', path, accessToken: accessToken);
  }

  Future<dynamic> post(String path, {Object? body, String? accessToken}) {
    return _send('POST', path, body: body, accessToken: accessToken);
  }

  Future<dynamic> patch(String path, {Object? body, String? accessToken}) {
    return _send('PATCH', path, body: body, accessToken: accessToken);
  }

  Future<dynamic> delete(String path, {String? accessToken}) {
    return _send('DELETE', path, accessToken: accessToken);
  }

  Future<dynamic> _send(
    String method,
    String path, {
    Object? body,
    String? accessToken,
  }) async {
    final uri = _buildUri(path);
    final headers = _headers(accessToken);

    if (body != null) {
      headers['Content-Type'] = 'application/json';
    }

    late final http.Response response;

    try {
      switch (method) {
        case 'GET':
          response = await _httpClient.get(uri, headers: headers);
        case 'POST':
          response = await _httpClient.post(
            uri,
            headers: headers,
            body: body == null ? null : jsonEncode(body),
          );
        case 'PATCH':
          response = await _httpClient.patch(
            uri,
            headers: headers,
            body: body == null ? null : jsonEncode(body),
          );
        case 'DELETE':
          response = await _httpClient.delete(uri, headers: headers);
        default:
          throw ArgumentError.value(method, 'method', 'Unsupported method');
      }
    } on ApiException {
      rethrow;
    } catch (error) {
      throw const ApiException(
        statusCode: 0,
        message: 'Unable to connect to the server. Check your connection.',
      );
    }

    return _handleResponse(response);
  }

  Future<dynamic> postMultipart(
    String path, {
    required String fieldName,
    required String filename,
    required List<int> bytes,
    Map<String, String>? fields,
    String? accessToken,
  }) async {
    final uri = _buildUri(path);
    final headers = _headers(accessToken);

    final request = http.MultipartRequest('POST', uri);
    request.headers.addAll(headers);

    if (fields != null) {
      request.fields.addAll(fields);
    }

    request.files.add(
      http.MultipartFile.fromBytes(fieldName, bytes, filename: filename),
    );

    late final http.StreamedResponse streamedResponse;
    try {
      streamedResponse = await _httpClient.send(request);
    } on ApiException {
      rethrow;
    } catch (e) {
      if (e is Error) rethrow;
      throw const ApiException(
        statusCode: 0,
        message: 'Unable to connect to the server. Check your connection.',
      );
    }

    final response = await http.Response.fromStream(streamedResponse);
    return _handleResponse(response);
  }

  dynamic _handleResponse(http.Response response) {
    final responseBody = response.body.trim();

    dynamic decodedBody;

    if (responseBody.isNotEmpty) {
      try {
        decodedBody = jsonDecode(responseBody);
      } on FormatException {
        if (response.statusCode >= 200 && response.statusCode < 300) {
          throw ApiException(
            statusCode: response.statusCode,
            message: 'The server returned an invalid response.',
          );
        }
      }
    }

    if (response.statusCode < 200 || response.statusCode >= 300) {
      String message;
      if (decodedBody is Map<String, dynamic>) {
        message =
            decodedBody['error']?.toString() ??
            decodedBody['message']?.toString() ??
            responseBody;
      } else if (responseBody.isNotEmpty) {
        message = responseBody;
      } else {
        message = 'Request failed with status ${response.statusCode}.';
      }

      throw ApiException(statusCode: response.statusCode, message: message);
    }

    return decodedBody;
  }

  void close() {
    _httpClient.close();
  }
}
