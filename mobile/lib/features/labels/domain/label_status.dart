import 'package:freezed_annotation/freezed_annotation.dart';

part 'label_status.freezed.dart';
part 'label_status.g.dart';

/// Possible processing states of a label.
enum LabelStatusValue {
  @JsonValue('pending') pending,
  @JsonValue('processing') processing,
  @JsonValue('ready') ready,
  @JsonValue('confirmed') confirmed,
  @JsonValue('failed') failed,
}

@freezed
class LabelStatus with _$LabelStatus {
  const factory LabelStatus({
    required String id,
    required LabelStatusValue status,
    @JsonKey(name: 'progress_pct') @Default(0) int progressPct,
    @JsonKey(name: 'error_message') String? errorMessage,
  }) = _LabelStatus;

  factory LabelStatus.fromJson(Map<String, dynamic> json) =>
      _$LabelStatusFromJson(json);
}
