import 'package:flutter/material.dart';

import '../../../core/network/api_client.dart';
import '../../../core/storage/token_storage.dart';
import '../../auth/models/user.dart';
import '../../groups/models/group.dart';
import '../../subgroups/models/subgroup.dart';
import '../controllers/access_periods_controller.dart';
import '../models/access_period.dart';
import '../services/access_periods_service.dart';
import '../widgets/access_period_card.dart';
import '../widgets/create_access_period_sheet.dart';

class SubgroupAccessPeriodsScreen extends StatefulWidget {
  const SubgroupAccessPeriodsScreen({
    required this.user,
    required this.parentGroup,
    required this.subgroup,
    this.controller,
    this.autoLoad = true,
    super.key,
  });

  final User user;
  final Group parentGroup;
  final Subgroup subgroup;
  final AccessPeriodsController? controller;
  final bool autoLoad;

  @override
  State<SubgroupAccessPeriodsScreen> createState() =>
      _SubgroupAccessPeriodsScreenState();
}

class _SubgroupAccessPeriodsScreenState
    extends State<SubgroupAccessPeriodsScreen> {
  late final AccessPeriodsController _controller;

  @override
  void initState() {
    super.initState();
    _controller =
        widget.controller ??
        AccessPeriodsController(
          accessPeriodsService: AccessPeriodsService(
            apiClient: ApiClient(),
            tokenStorage: TokenStorage(),
          ),
          groupId: widget.subgroup.groupId,
          subgroupId: widget.subgroup.id,
        );

    if (widget.autoLoad && widget.user.isFaculty) {
      _controller.loadAccessPeriods();
    }
  }

  void _openCreateSheet() {
    showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (_) => CreateAccessPeriodSheet(controller: _controller),
    ).then((created) {
      if (created == true && mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: const Text('Collection window created successfully.'),
            backgroundColor: Colors.green.shade800,
          ),
        );
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    // Strict role check: faculty and college admin only
    if (!widget.user.isFaculty) {
      return Scaffold(
        appBar: AppBar(title: const Text('Collection Windows')),
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(32),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(Icons.lock_rounded, size: 48, color: colors.error),
                const SizedBox(height: 16),
                Text(
                  'Faculty Access Only',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  'Collection window management is strictly restricted to authorized faculty members.',
                  textAlign: TextAlign.center,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: colors.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    }

    return Scaffold(
      appBar: AppBar(
        title: Text(widget.subgroup.name),
        actions: [
          IconButton(
            key: const Key('refresh_access_periods_button'),
            icon: const Icon(Icons.refresh_rounded),
            tooltip: 'Refresh access periods',
            onPressed: () => _controller.refresh(),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        key: const Key('add_access_period_button'),
        onPressed: _openCreateSheet,
        icon: const Icon(Icons.add_rounded),
        label: const Text('New Window'),
      ),
      body: SafeArea(
        child: Column(
          children: [
            // Context header banner
            Container(
              width: double.infinity,
              margin: const EdgeInsets.fromLTRB(16, 8, 16, 12),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: colors.primaryContainer.withValues(alpha: 0.35),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(
                  color: colors.primary.withValues(alpha: 0.2),
                ),
              ),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Icon(
                    Icons.event_note_rounded,
                    color: colors.primary,
                    size: 20,
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Parent Group: ${widget.parentGroup.name}',
                          style: theme.textTheme.labelMedium?.copyWith(
                            fontWeight: FontWeight.bold,
                            color: colors.primary,
                          ),
                        ),
                        const SizedBox(height: 2),
                        Text(
                          'Managing collection windows for subgroup #${widget.subgroup.id}',
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: colors.onPrimaryContainer,
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),

            // Content Area
            Expanded(
              child: ListenableBuilder(
                listenable: _controller,
                builder: (context, _) {
                  return switch (_controller.state) {
                    AccessPeriodsInitial() => const SizedBox.shrink(),
                    AccessPeriodsLoading() => const Center(
                      child: CircularProgressIndicator(
                        key: Key('access_periods_loading_indicator'),
                      ),
                    ),
                    AccessPeriodsLoaded(:final periods) => _buildPeriodsList(
                      context,
                      periods,
                    ),
                    AccessPeriodsEmpty() => _buildEmptyState(context),
                    AccessPeriodsError(:final message, :final canRetry) =>
                      _buildErrorState(context, message, canRetry),
                  };
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildPeriodsList(BuildContext context, List<AccessPeriod> periods) {
    return RefreshIndicator(
      onRefresh: () => _controller.refresh(),
      child: ListView.separated(
        padding: const EdgeInsets.fromLTRB(16, 8, 16, 88),
        itemCount: periods.length,
        separatorBuilder: (context, _) => const SizedBox(height: 12),
        itemBuilder: (context, index) {
          final period = periods[index];
          return AccessPeriodCard(period: period);
        },
      ),
    );
  }

  Widget _buildEmptyState(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    return Center(
      key: const Key('access_periods_empty_state'),
      child: SingleChildScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Container(
              width: 72,
              height: 72,
              decoration: BoxDecoration(
                color: colors.surfaceContainerHighest,
                shape: BoxShape.circle,
              ),
              child: Icon(
                Icons.calendar_month_outlined,
                size: 40,
                color: colors.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 20),
            Text(
              'No Collection Windows Configured',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.bold,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              'No active or scheduled access periods exist for this subgroup. Students cannot submit documents until a collection window is defined.',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: colors.onSurfaceVariant,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 20),
            FilledButton.icon(
              onPressed: _openCreateSheet,
              icon: const Icon(Icons.add_rounded, size: 18),
              label: const Text('Add Window'),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildErrorState(BuildContext context, String message, bool canRetry) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    return Center(
      key: const Key('access_periods_error_state'),
      child: SingleChildScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Container(
              width: 72,
              height: 72,
              decoration: BoxDecoration(
                color: colors.errorContainer,
                shape: BoxShape.circle,
              ),
              child: Icon(
                Icons.error_outline_rounded,
                size: 36,
                color: colors.onErrorContainer,
              ),
            ),
            const SizedBox(height: 18),
            Text(
              'Unable to load collection windows',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.bold,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              message,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: colors.onSurfaceVariant,
              ),
              textAlign: TextAlign.center,
            ),
            if (canRetry) ...[
              const SizedBox(height: 20),
              FilledButton.icon(
                onPressed: () => _controller.refresh(),
                icon: const Icon(Icons.refresh_rounded, size: 18),
                label: const Text('Try Again'),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
