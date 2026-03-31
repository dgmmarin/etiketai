// ignore_for_file: constant_identifier_names

/// URL constants for the EtiketAI API.
class ApiEndpoints {
  ApiEndpoints._();

  // Base URL — 10.0.2.2 is the Android emulator loopback to the host machine.
  // Override via environment or build flavour when targeting real devices.
  static const String baseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://10.0.2.2:8080',
  );

  // Auth
  static const String login = '/v1/auth/login';
  static const String register = '/v1/auth/register';
  static const String refresh = '/v1/auth/refresh';
  static const String logout = '/v1/auth/logout';

  // Labels
  static const String labels = '/v1/labels';
  static String label(String id) => '/v1/labels/$id';
  static String labelStatus(String id) => '/v1/labels/$id/status';
  static const String labelUpload = '/v1/labels/upload';
  static String labelConfirm(String id) => '/v1/labels/$id/confirm';

  // Workspace
  static const String workspaceProfile = '/v1/workspace/profile';
}
