import 'package:freezed_annotation/freezed_annotation.dart';

import 'label_status.dart';

part 'label.freezed.dart';
part 'label.g.dart';

/// Lightweight model used in the list view.
@freezed
class LabelListItem with _$LabelListItem {
  const factory LabelListItem({
    required String id,
    @JsonKey(name: 'product_name') String? productName,
    @JsonKey(name: 'thumbnail_url') String? thumbnailUrl,
    required LabelStatusValue status,
    @JsonKey(name: 'created_at') required DateTime createdAt,
    @JsonKey(name: 'updated_at') required DateTime updatedAt,
  }) = _LabelListItem;

  factory LabelListItem.fromJson(Map<String, dynamic> json) =>
      _$LabelListItemFromJson(json);
}
