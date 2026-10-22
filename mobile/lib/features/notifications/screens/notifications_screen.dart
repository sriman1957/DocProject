import 'package:flutter/material.dart';

import '../../../core/network/api_client.dart';
import '../../../core/storage/token_storage.dart';
import '../controllers/notifications_controller.dart';
import '../models/notification_item.dart';
import '../services/notifications_service.dart';
import '../widgets/notification_tile.dart';

class NotificationsScreen extends StatefulWidget {
  const NotificationsScreen({this.controller, this.autoLoad = true, super.key});

  final NotificationsController? controller;
  final bool autoLoad;

  @override
  State<NotificationsScreen> createState() => _NotificationsScreenState();
}

class _NotificationsScreenState extends State<NotificationsScreen> {
  late final NotificationsController _controller;

  @override
  void initState() {
    super.initState();
    _controller =
        widget.controller ??
        NotificationsController(
          notificationsService: NotificationsService(
            apiClient: ApiClient(),
            tokenStorage: TokenStorage(),
          ),
        );

    if (widget.autoLoad) {
      _controller.loadNotifications();
    }
  }

  Future<void> _handleMarkAsRead(NotificationItem item) async {
    try {
      await _controller.markAsRead(item.id);
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Failed to update notification status.')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Notifications'),
        actions: [
          IconButton(
            key: const Key('refresh_notifications_button'),
            icon: const Icon(Icons.refresh_rounded),
            tooltip: 'Refresh notifications',
            onPressed: () => _controller.refresh(),
          ),
        ],
      ),
      body: SafeArea(
        child: ListenableBuilder(
          listenable: _controller,
          builder: (context, _) {
            return switch (_controller.state) {
              NotificationsInitial() => const SizedBox.shrink(),
              NotificationsLoading() => const Center(
                child: CircularProgressIndicator(
                  key: Key('notifications_loading_indicator'),
                ),
              ),
              NotificationsLoaded(:final notifications, :final unreadCount) =>
                _buildLoadedState(context, notifications, unreadCount),
              NotificationsEmpty() => _buildEmptyState(context),
              NotificationsError(:final message, :final canRetry) =>
                _buildErrorState(context, message, canRetry),
            };
          },
        ),
      ),
    );
  }

  Widget _buildLoadedState(
    BuildContext context,
    List<NotificationItem> notifications,
    int unreadCount,
  ) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    return RefreshIndicator(
      onRefresh: () => _controller.refresh(),
      child: Column(
        children: [
          if (unreadCount > 0)
            Container(
              width: double.infinity,
              margin: const EdgeInsets.fromLTRB(16, 8, 16, 8),
              padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
              decoration: BoxDecoration(
                color: colors.primaryContainer.withValues(alpha: 0.4),
                borderRadius: BorderRadius.circular(10),
              ),
              child: Row(
                children: [
                  Icon(
                    Icons.mark_email_unread_outlined,
                    size: 18,
                    color: colors.primary,
                  ),
                  const SizedBox(width: 8),
                  Text(
                    '$unreadCount unread notification${unreadCount == 1 ? '' : 's'}',
                    style: theme.textTheme.labelMedium?.copyWith(
                      color: colors.onPrimaryContainer,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ],
              ),
            ),
          Expanded(
            child: ListView.separated(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
              itemCount: notifications.length,
              separatorBuilder: (context, _) => const SizedBox(height: 10),
              itemBuilder: (context, index) {
                final item = notifications[index];
                return NotificationTile(
                  notification: item,
                  onMarkAsRead: () => _handleMarkAsRead(item),
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildEmptyState(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    return Center(
      key: const Key('notifications_empty_state'),
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
                Icons.notifications_none_rounded,
                size: 40,
                color: colors.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 20),
            Text(
              'No Notifications Yet',
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.bold,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              'You are all caught up! Updates regarding your groups, submissions, and access periods will appear here.',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: colors.onSurfaceVariant,
              ),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 20),
            OutlinedButton.icon(
              onPressed: () => _controller.refresh(),
              icon: const Icon(Icons.refresh_rounded, size: 18),
              label: const Text('Refresh'),
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
      key: const Key('notifications_error_state'),
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
              'Unable to load notifications',
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
