import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/api/api_client.dart';
import '../../../core/api/api_endpoints.dart';
import '../../../core/storage/secure_storage.dart';
import '../domain/auth_user.dart';

/// Result of a successful authentication call.
class AuthResult {
  const AuthResult({
    required this.accessToken,
    required this.refreshToken,
    required this.user,
  });

  final String accessToken;
  final String refreshToken;
  final AuthUser user;
}

class AuthRepository {
  AuthRepository(this._dio, this._storage);

  final Dio _dio;
  final SecureStorageService _storage;

  /// Login with email + password.
  Future<AuthResult> login(String email, String password) async {
    final resp = await _dio.post(
      ApiEndpoints.login,
      data: {'email': email, 'password': password},
    );
    return _parseAuthResponse(resp.data as Map<String, dynamic>);
  }

  /// Register a new workspace + admin user.
  Future<AuthResult> register({
    required String email,
    required String password,
    required String workspaceName,
    required String cui,
  }) async {
    final resp = await _dio.post(
      ApiEndpoints.register,
      data: {
        'email': email,
        'password': password,
        'workspace_name': workspaceName,
        'cui': cui,
      },
    );
    return _parseAuthResponse(resp.data as Map<String, dynamic>);
  }

  /// Silent token refresh using the stored refresh token.
  Future<AuthResult> refresh(String refreshToken) async {
    final resp = await _dio.post(
      ApiEndpoints.refresh,
      data: {'refresh_token': refreshToken},
    );
    return _parseAuthResponse(resp.data as Map<String, dynamic>);
  }

  /// Logout — invalidates the refresh token server-side.
  Future<void> logout(String refreshToken) async {
    try {
      await _dio.post(
        ApiEndpoints.logout,
        data: {'refresh_token': refreshToken},
      );
    } catch (_) {
      // Best-effort; always clear local state regardless.
    }
    await _storage.deleteRefreshToken();
  }

  // ---------------------------------------------------------------------------
  // Helpers
  // ---------------------------------------------------------------------------

  AuthResult _parseAuthResponse(Map<String, dynamic> data) {
    final accessToken = data['access_token'] as String;
    final refreshToken = data['refresh_token'] as String;
    final userJson = data['user'] as Map<String, dynamic>;
    return AuthResult(
      accessToken: accessToken,
      refreshToken: refreshToken,
      user: AuthUser.fromJson(userJson),
    );
  }
}

final authRepositoryProvider = Provider<AuthRepository>((ref) {
  return AuthRepository(
    ref.watch(dioProvider),
    ref.watch(secureStorageProvider),
  );
});
