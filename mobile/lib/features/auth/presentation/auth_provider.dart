import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/api/api_client.dart';
import '../../../core/storage/secure_storage.dart';
import '../data/auth_repository.dart';
import '../domain/auth_state.dart';

class AuthNotifier extends StateNotifier<AuthState> {
  AuthNotifier(this._repo, this._storage, this._ref)
      : super(const AuthState.loading()) {
    _tryAutoLogin();
  }

  final AuthRepository _repo;
  final SecureStorageService _storage;
  final Ref _ref;

  /// Attempt silent login using a stored refresh token.
  Future<void> _tryAutoLogin() async {
    final rt = await _storage.readRefreshToken();
    if (rt == null) {
      state = const AuthState.unauthenticated();
      return;
    }
    try {
      final result = await _repo.refresh(rt);
      _ref.read(accessTokenProvider.notifier).set(result.accessToken);
      await _storage.saveRefreshToken(result.refreshToken);
      state = AuthState.authenticated(
        user: result.user,
        accessToken: result.accessToken,
      );
    } catch (_) {
      await _storage.deleteRefreshToken();
      state = const AuthState.unauthenticated();
    }
  }

  Future<void> login(String email, String password) async {
    state = const AuthState.loading();
    try {
      final result = await _repo.login(email, password);
      _ref.read(accessTokenProvider.notifier).set(result.accessToken);
      await _storage.saveRefreshToken(result.refreshToken);
      state = AuthState.authenticated(
        user: result.user,
        accessToken: result.accessToken,
      );
    } catch (e) {
      state = const AuthState.unauthenticated();
      rethrow;
    }
  }

  Future<void> register({
    required String email,
    required String password,
    required String workspaceName,
    required String cui,
  }) async {
    state = const AuthState.loading();
    try {
      final result = await _repo.register(
        email: email,
        password: password,
        workspaceName: workspaceName,
        cui: cui,
      );
      _ref.read(accessTokenProvider.notifier).set(result.accessToken);
      await _storage.saveRefreshToken(result.refreshToken);
      state = AuthState.authenticated(
        user: result.user,
        accessToken: result.accessToken,
      );
    } catch (e) {
      state = const AuthState.unauthenticated();
      rethrow;
    }
  }

  Future<void> logout() async {
    final rt = await _storage.readRefreshToken();
    if (rt != null) {
      await _repo.logout(rt);
    }
    _ref.read(accessTokenProvider.notifier).clear();
    state = const AuthState.unauthenticated();
  }
}

final authProvider = StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  return AuthNotifier(
    ref.watch(authRepositoryProvider),
    ref.watch(secureStorageProvider),
    ref,
  );
});
