import 'package:freezed_annotation/freezed_annotation.dart';

import 'auth_user.dart';

part 'auth_state.freezed.dart';

@freezed
class AuthState with _$AuthState {
  /// Initial state — checking stored credentials.
  const factory AuthState.loading() = AuthStateLoading;

  /// User is logged in with a valid access token.
  const factory AuthState.authenticated({
    required AuthUser user,
    required String accessToken,
  }) = AuthStateAuthenticated;

  /// No valid session exists.
  const factory AuthState.unauthenticated() = AuthStateUnauthenticated;
}
