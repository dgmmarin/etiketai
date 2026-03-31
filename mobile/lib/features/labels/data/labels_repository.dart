import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/api/api_client.dart';
import '../../../core/api/api_endpoints.dart';
import '../domain/label.dart';
import '../domain/label_detail.dart';
import '../domain/label_status.dart';

class LabelsRepository {
  LabelsRepository(this._dio);

  final Dio _dio;

  /// List labels with optional filters.
  Future<List<LabelListItem>> list({
    String? search,
    LabelStatusValue? status,
    int page = 1,
    int pageSize = 20,
  }) async {
    final resp = await _dio.get(
      ApiEndpoints.labels,
      queryParameters: {
        if (search != null && search.isNotEmpty) 'search': search,
        if (status != null) 'status': status.name,
        'page': page,
        'page_size': pageSize,
      },
    );
    final items = resp.data['items'] as List<dynamic>;
    return items
        .map((e) => LabelListItem.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  /// Get full label detail by id.
  Future<LabelDetail> get(String id) async {
    final resp = await _dio.get(ApiEndpoints.label(id));
    return LabelDetail.fromJson(resp.data as Map<String, dynamic>);
  }

  /// Poll status of a label.
  Future<LabelStatus> getStatus(String id) async {
    final resp = await _dio.get(ApiEndpoints.labelStatus(id));
    return LabelStatus.fromJson(resp.data as Map<String, dynamic>);
  }

  /// Upload a photo file and create a new label job.
  Future<LabelDetail> upload(File file) async {
    final formData = FormData.fromMap({
      'file': await MultipartFile.fromFile(
        file.path,
        filename: file.path.split('/').last,
      ),
    });
    final resp = await _dio.post(
      ApiEndpoints.labelUpload,
      data: formData,
      options: Options(contentType: 'multipart/form-data'),
    );
    return LabelDetail.fromJson(resp.data as Map<String, dynamic>);
  }

  /// Update translated / editable fields of a label.
  Future<LabelDetail> updateFields(
      String id, Map<String, String> fields) async {
    final resp = await _dio.patch(
      ApiEndpoints.label(id),
      data: {'fields': fields},
    );
    return LabelDetail.fromJson(resp.data as Map<String, dynamic>);
  }

  /// Confirm a label (locks it from further editing).
  Future<LabelDetail> confirm(String id) async {
    final resp = await _dio.post(ApiEndpoints.labelConfirm(id));
    return LabelDetail.fromJson(resp.data as Map<String, dynamic>);
  }

  /// Delete a label.
  Future<void> delete(String id) async {
    await _dio.delete(ApiEndpoints.label(id));
  }
}

final labelsRepositoryProvider = Provider<LabelsRepository>((ref) {
  return LabelsRepository(ref.watch(dioProvider));
});
