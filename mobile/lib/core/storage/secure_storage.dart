import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

const _kRefreshTokenKey = 'refresh_token';

/// Thin wrapper around [FlutterSecureStorage] for typed token persistence.
class SecureStorageService {
  SecureStorageService(this._storage);

  final FlutterSecureStorage _storage;

  Future<void> saveRefreshToken(String token) =>
      _storage.write(key: _kRefreshTokenKey, value: token);

  Future<String?> readRefreshToken() =>
      _storage.read(key: _kRefreshTokenKey);

  Future<void> deleteRefreshToken() =>
      _storage.delete(key: _kRefreshTokenKey);

  Future<void> clearAll() => _storage.deleteAll();
}

final secureStorageProvider = Provider<SecureStorageService>((ref) {
  const storage = FlutterSecureStorage(
    aOptions: AndroidOptions(encryptedSharedPreferences: true),
    iOptions: IOSOptions(accessibility: KeychainAccessibility.first_unlock),
  );
  return SecureStorageService(storage);
});
