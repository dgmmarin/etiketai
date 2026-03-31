import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../data/workspace_repository.dart';
import '../domain/workspace.dart';

final workspaceProfileProvider =
    FutureProvider<WorkspaceProfile>((ref) async {
  final repo = ref.watch(workspaceRepositoryProvider);
  return repo.getProfile();
});
