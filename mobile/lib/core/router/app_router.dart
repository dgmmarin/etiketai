import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../features/auth/domain/auth_state.dart';
import '../../features/auth/presentation/auth_provider.dart';
import '../../features/auth/presentation/login_screen.dart';
import '../../features/auth/presentation/register_screen.dart';
import '../../features/auth/presentation/splash_screen.dart';
import '../../features/labels/presentation/camera_screen.dart';
import '../../features/labels/presentation/label_detail_screen.dart';
import '../../features/labels/presentation/labels_screen.dart';
import '../../features/workspace/presentation/workspace_screen.dart';

final routerProvider = Provider<GoRouter>((ref) {
  final authNotifier = ref.watch(authProvider.notifier);

  return GoRouter(
    initialLocation: '/',
    refreshListenable: _AuthListenable(ref),
    redirect: (context, state) {
      final authState = ref.read(authProvider);
      final isLoading = authState is AuthStateLoading;
      final isAuthenticated = authState is AuthStateAuthenticated;

      final onSplash = state.matchedLocation == '/';
      final onAuth = state.matchedLocation == '/login' ||
          state.matchedLocation == '/register';

      if (isLoading) {
        return onSplash ? null : '/';
      }

      if (!isAuthenticated && !onAuth) {
        return '/login';
      }

      if (isAuthenticated && onAuth) {
        return '/labels';
      }

      return null;
    },
    routes: [
      GoRoute(
        path: '/',
        builder: (_, __) => const SplashScreen(),
      ),
      GoRoute(
        path: '/login',
        builder: (_, __) => const LoginScreen(),
      ),
      GoRoute(
        path: '/register',
        builder: (_, __) => const RegisterScreen(),
      ),
      GoRoute(
        path: '/labels',
        builder: (_, __) => const LabelsScreen(),
      ),
      GoRoute(
        path: '/labels/:id',
        builder: (_, state) => LabelDetailScreen(
          labelId: state.pathParameters['id']!,
        ),
      ),
      GoRoute(
        path: '/camera',
        builder: (_, __) => const CameraScreen(),
      ),
      GoRoute(
        path: '/workspace',
        builder: (_, __) => const WorkspaceScreen(),
      ),
    ],
    errorBuilder: (context, state) => Scaffold(
      body: Center(
        child: Text('Pagina nu a fost găsită: ${state.error}'),
      ),
    ),
  );
});

/// A [Listenable] that fires whenever [authProvider] changes,
/// so [GoRouter.refreshListenable] re-evaluates the redirect guard.
class _AuthListenable extends ChangeNotifier {
  _AuthListenable(Ref ref) {
    ref.listen<AuthState>(authProvider, (_, __) => notifyListeners());
  }
}
