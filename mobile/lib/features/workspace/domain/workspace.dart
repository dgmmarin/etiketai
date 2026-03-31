import 'package:freezed_annotation/freezed_annotation.dart';

part 'workspace.freezed.dart';
part 'workspace.g.dart';

@freezed
class WorkspaceProfile with _$WorkspaceProfile {
  const factory WorkspaceProfile({
    required String id,
    required String name,
    @JsonKey(name: 'cui') required String cui,
    @JsonKey(name: 'plan') required String plan,
    @JsonKey(name: 'labels_used') required int labelsUsed,
    @JsonKey(name: 'labels_limit') required int labelsLimit,
    @JsonKey(name: 'created_at') required DateTime createdAt,
    @JsonKey(name: 'trial_ends_at') DateTime? trialEndsAt,
  }) = _WorkspaceProfile;

  factory WorkspaceProfile.fromJson(Map<String, dynamic> json) =>
      _$WorkspaceProfileFromJson(json);
}
