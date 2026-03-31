import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../storage/secure_storage.dart';
import 'api_endpoints.dart';

// ---------------------------------------------------------------------------
// In-memory access token holder
// ---------------------------------------------------------------------------

/// Holds the current access token in memory only (never persisted to disk).
class AccessTokenNotifier extends StateNotifier<String?> {
  AccessTokenNotifier() : super(null);

  void set(String token) => state = token;
  void clear() => state = null;
}

final accessTokenProvider =
    StateNotifierProvider<AccessTokenNotifier, String?>((ref) {
  return AccessTokenNotifier();
});

// ---------------------------------------------------------------------------
// Auth interceptor
// ---------------------------------------------------------------------------

class _AuthInterceptor extends Interceptor {
  _AuthInterceptor(this._ref);

  final Ref _ref;

  // Flag to prevent concurrent refresh loops
  bool _isRefreshing = false;

  @override
  void onRequest(
      RequestOptions options, RequestInterceptorHandler handler) async {
    final token = _ref.read(accessTokenProvider);
    if (token != null) {
      options.headers['Authorization'] = 'Bearer $token';
    }
    handler.next(options);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) async {
    if (err.response?.statusCode == 401 && !_isRefreshing) {
      _isRefreshing = true;
      try {
        final storage = _ref.read(secureStorageProvider);
        final refreshToken = await storage.readRefreshToken();
        if (refreshToken == null) {
          _clearAndReject(err, handler);
          return;
        }

        // Perform refresh using a clean Dio instance (no interceptors) to
        // avoid recursive loops.
        final plainDio = Dio(BaseOptions(baseUrl: ApiEndpoints.baseUrl));
        final refreshResp = await plainDio.post(
          ApiEndpoints.refresh,
          data: {'refresh_token': refreshToken},
        );

        final newAccessToken =
            refreshResp.data['access_token'] as String?;
        final newRefreshToken =
            refreshResp.data['refresh_token'] as String?;

        if (newAccessToken == null) {
          _clearAndReject(err, handler);
          return;
        }

        _ref.read(accessTokenProvider.notifier).set(newAccessToken);
        if (newRefreshToken != null) {
          await storage.saveRefreshToken(newRefreshToken);
        }

        // Retry the original request with the new token
        final opts = err.requestOptions;
        opts.headers['Authorization'] = 'Bearer $newAccessToken';
        final retryResp = await plainDio.fetch(opts);
        handler.resolve(retryResp);
      } catch (_) {
        _clearAndReject(err, handler);
      } finally {
        _isRefreshing = false;
      }
      return;
    }
    handler.next(err);
  }

  void _clearAndReject(DioException err, ErrorInterceptorHandler handler) {
    _ref.read(accessTokenProvider.notifier).clear();
    _ref.read(secureStorageProvider).deleteRefreshToken();
    handler.next(err);
  }
}

// ---------------------------------------------------------------------------
// Dio provider
// ---------------------------------------------------------------------------

final dioProvider = Provider<Dio>((ref) {
  final dio = Dio(
    BaseOptions(
      baseUrl: ApiEndpoints.baseUrl,
      connectTimeout: const Duration(seconds: 15),
      receiveTimeout: const Duration(seconds: 30),
      headers: {'Content-Type': 'application/json'},
    ),
  );

  dio.interceptors.add(_AuthInterceptor(ref));

  return dio;
});
