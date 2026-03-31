import 'package:freezed_annotation/freezed_annotation.dart';

import 'label_status.dart';

part 'label_detail.freezed.dart';
part 'label_detail.g.dart';

/// A single translated field extracted from a label.
@freezed
class LabelField with _$LabelField {
  const factory LabelField({
    @JsonKey(name: 'key') required String fieldKey,
    @JsonKey(name: 'original_text') String? originalText,
    @JsonKey(name: 'translated_text') String? translatedText,
    @JsonKey(name: 'field_type') String? fieldType,
  }) = _LabelField;

  factory LabelField.fromJson(Map<String, dynamic> json) =>
      _$LabelFieldFromJson(json);
}

/// Compliance check result.
@freezed
class ComplianceCheck with _$ComplianceCheck {
  const factory ComplianceCheck({
    @JsonKey(name: 'is_compliant') required bool isCompliant,
    required List<String> issues,
    required List<String> warnings,
  }) = _ComplianceCheck;

  factory ComplianceCheck.fromJson(Map<String, dynamic> json) =>
      _$ComplianceCheckFromJson(json);
}

/// Full label detail model.
@freezed
class LabelDetail with _$LabelDetail {
  const factory LabelDetail({
    required String id,
    @JsonKey(name: 'product_name') String? productName,
    @JsonKey(name: 'image_url') String? imageUrl,
    @JsonKey(name: 'thumbnail_url') String? thumbnailUrl,
    required LabelStatusValue status,
    @Default([]) List<LabelField> fields,
    @JsonKey(name: 'compliance_check') ComplianceCheck? complianceCheck,
    @JsonKey(name: 'source_language') String? sourceLanguage,
    @JsonKey(name: 'created_at') required DateTime createdAt,
    @JsonKey(name: 'updated_at') required DateTime updatedAt,
    @JsonKey(name: 'confirmed_at') DateTime? confirmedAt,
  }) = _LabelDetail;

  factory LabelDetail.fromJson(Map<String, dynamic> json) =>
      _$LabelDetailFromJson(json);
}
