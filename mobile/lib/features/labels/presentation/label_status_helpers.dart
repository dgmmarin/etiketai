import 'package:flutter/material.dart';

import '../../../core/theme/app_theme.dart';
import '../domain/label_status.dart';

Color statusColor(LabelStatusValue s) {
  return switch (s) {
    LabelStatusValue.pending => AppColors.statusPending,
    LabelStatusValue.processing => AppColors.statusProcessing,
    LabelStatusValue.ready => AppColors.statusReady,
    LabelStatusValue.confirmed => AppColors.statusConfirmed,
    LabelStatusValue.failed => AppColors.statusFailed,
  };
}

String statusLabel(LabelStatusValue s) {
  return switch (s) {
    LabelStatusValue.pending => 'În așteptare',
    LabelStatusValue.processing => 'Procesare',
    LabelStatusValue.ready => 'Gata',
    LabelStatusValue.confirmed => 'Confirmat',
    LabelStatusValue.failed => 'Eroare',
  };
}
