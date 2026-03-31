import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../core/theme/app_theme.dart';
import '../../auth/presentation/auth_provider.dart';
import '../domain/workspace.dart';
import 'workspace_provider.dart';

class WorkspaceScreen extends ConsumerWidget {
  const WorkspaceScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(workspaceProfileProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Profil spațiu de lucru'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.go('/labels'),
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.logout),
            tooltip: 'Deconectare',
            onPressed: () => _confirmLogout(context, ref),
          ),
        ],
      ),
      body: profileAsync.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.error_outline, size: 48, color: Colors.black38),
              const SizedBox(height: 12),
              const Text('Nu s-au putut încărca datele.',
                  style: TextStyle(color: Colors.black54)),
              const SizedBox(height: 16),
              ElevatedButton(
                onPressed: () => ref.invalidate(workspaceProfileProvider),
                child: const Text('Reîncercați'),
              ),
            ],
          ),
        ),
        data: (profile) => _ProfileBody(profile: profile),
      ),
    );
  }

  Future<void> _confirmLogout(BuildContext context, WidgetRef ref) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: const Text('Deconectare'),
        content: const Text('Sigur doriți să vă deconectați?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Anulare'),
          ),
          ElevatedButton(
            onPressed: () => Navigator.pop(context, true),
            style: ElevatedButton.styleFrom(
              backgroundColor: AppColors.error,
            ),
            child: const Text('Deconectare'),
          ),
        ],
      ),
    );
    if (confirmed == true) {
      await ref.read(authProvider.notifier).logout();
      // Router redirect handles navigation to /login
    }
  }
}

class _ProfileBody extends StatelessWidget {
  const _ProfileBody({required this.profile});

  final WorkspaceProfile profile;

  String _planLabel(String plan) {
    return switch (plan) {
      'free' => 'Gratuit',
      'starter' => 'Starter',
      'pro' => 'Pro',
      'enterprise' => 'Enterprise',
      _ => plan,
    };
  }

  @override
  Widget build(BuildContext context) {
    final df = DateFormat('dd.MM.yyyy', 'ro_RO');
    final usagePct =
        profile.labelsLimit > 0 ? profile.labelsUsed / profile.labelsLimit : 0.0;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Company header
          Center(
            child: Column(
              children: [
                CircleAvatar(
                  radius: 40,
                  backgroundColor: AppColors.primary.withValues(alpha: 0.15),
                  child: Text(
                    profile.name.isNotEmpty
                        ? profile.name[0].toUpperCase()
                        : '?',
                    style: const TextStyle(
                        fontSize: 36,
                        fontWeight: FontWeight.bold,
                        color: AppColors.primary),
                  ),
                ),
                const SizedBox(height: 12),
                Text(
                  profile.name,
                  style: Theme.of(context)
                      .textTheme
                      .headlineSmall
                      ?.copyWith(fontWeight: FontWeight.bold),
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 4),
                Text(
                  'CUI: ${profile.cui}',
                  style: const TextStyle(color: Colors.black54),
                ),
              ],
            ),
          ),

          const SizedBox(height: 28),
          const Divider(),
          const SizedBox(height: 16),

          // Plan info
          _InfoSection(
            title: 'Abonament',
            children: [
              _InfoRow(
                icon: Icons.star_outline,
                label: 'Plan',
                value: _planLabel(profile.plan),
              ),
              _InfoRow(
                icon: Icons.calendar_today_outlined,
                label: 'Cont creat',
                value: df.format(profile.createdAt.toLocal()),
              ),
              if (profile.trialEndsAt != null)
                _InfoRow(
                  icon: Icons.hourglass_empty_outlined,
                  label: 'Trial până la',
                  value: df.format(profile.trialEndsAt!.toLocal()),
                  valueColor: AppColors.warning,
                ),
            ],
          ),

          const SizedBox(height: 20),

          // Usage
          _InfoSection(
            title: 'Utilizare etichete',
            children: [
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Text(
                          '${profile.labelsUsed} / ${profile.labelsLimit} etichete',
                          style: const TextStyle(fontWeight: FontWeight.w500),
                        ),
                        Text(
                          '${(usagePct * 100).toStringAsFixed(0)}%',
                          style: TextStyle(
                            color: usagePct > 0.9
                                ? AppColors.error
                                : Colors.black54,
                            fontWeight: FontWeight.w500,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    ClipRRect(
                      borderRadius: BorderRadius.circular(4),
                      child: LinearProgressIndicator(
                        value: usagePct.clamp(0.0, 1.0),
                        minHeight: 8,
                        backgroundColor: Colors.grey[200],
                        valueColor: AlwaysStoppedAnimation(
                          usagePct > 0.9
                              ? AppColors.error
                              : AppColors.primary,
                        ),
                      ),
                    ),
                    if (usagePct > 0.8)
                      Padding(
                        padding: const EdgeInsets.only(top: 8),
                        child: Text(
                          'Aproape de limita planului. Luați în considerare un upgrade.',
                          style: TextStyle(
                            fontSize: 12,
                            color: usagePct > 0.9
                                ? AppColors.error
                                : AppColors.warning,
                          ),
                        ),
                      ),
                  ],
                ),
              ),
            ],
          ),

          const SizedBox(height: 32),
        ],
      ),
    );
  }
}

class _InfoSection extends StatelessWidget {
  const _InfoSection({required this.title, required this.children});

  final String title;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          title,
          style: Theme.of(context)
              .textTheme
              .titleSmall
              ?.copyWith(
                  color: Colors.black45,
                  fontWeight: FontWeight.w600,
                  letterSpacing: 0.5),
        ),
        const SizedBox(height: 8),
        Card(
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 4),
            child: Column(children: children),
          ),
        ),
      ],
    );
  }
}

class _InfoRow extends StatelessWidget {
  const _InfoRow({
    required this.icon,
    required this.label,
    required this.value,
    this.valueColor,
  });

  final IconData icon;
  final String label;
  final String value;
  final Color? valueColor;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      child: Row(
        children: [
          Icon(icon, size: 20, color: Colors.black45),
          const SizedBox(width: 12),
          Text(label, style: const TextStyle(color: Colors.black54)),
          const Spacer(),
          Text(
            value,
            style: TextStyle(
              fontWeight: FontWeight.w500,
              color: valueColor,
            ),
          ),
        ],
      ),
    );
  }
}
