import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/api/api_client.dart';
import '../../../core/api/api_endpoints.dart';
import '../domain/workspace.dart';

class WorkspaceRepository {
  WorkspaceRepository(this._dio);

  final Dio _dio;

  Future<WorkspaceProfile> getProfile() async {
    final resp = await _dio.get(ApiEndpoints.workspaceProfile);
    return WorkspaceProfile.fromJson(resp.data as Map<String, dynamic>);
  }
}

final workspaceRepositoryProvider = Provider<WorkspaceRepository>((ref) {
  return WorkspaceRepository(ref.watch(dioProvider));
});
