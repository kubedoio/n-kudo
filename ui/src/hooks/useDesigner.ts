'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { authStore } from '@/store/auth';
import { CreateTemplateRequest, DeployTemplateRequest, Template } from '@/api/types';

const TEMPLATES_QUERY_KEY = 'templates';

function getClient() {
    const client = authStore.getState().client;
    if (!client) {
        throw new Error('API client not initialized');
    }
    return client;
}

// Query Hooks

export function useTemplates() {
    return useQuery({
        queryKey: [TEMPLATES_QUERY_KEY],
        queryFn: async () => {
            const client = getClient();
            return client.listTemplates();
        },
    });
}

export function useTemplate(templateId: string) {
    return useQuery({
        queryKey: [TEMPLATES_QUERY_KEY, templateId],
        queryFn: async () => {
            const client = getClient();
            return client.getTemplate(templateId);
        },
        enabled: !!templateId,
    });
}

// Mutation Hooks

export function useCreateTemplate() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (req: CreateTemplateRequest): Promise<Template> => {
            const client = getClient();
            return client.createTemplate(req);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: [TEMPLATES_QUERY_KEY] });
        },
    });
}

export function useUpdateTemplate() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async ({ templateId, req }: { templateId: string; req: CreateTemplateRequest }): Promise<Template> => {
            const client = getClient();
            return client.updateTemplate(templateId, req);
        },
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({ queryKey: [TEMPLATES_QUERY_KEY] });
            queryClient.invalidateQueries({ queryKey: [TEMPLATES_QUERY_KEY, variables.templateId] });
        },
    });
}

export function useDeleteTemplate() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (templateId: string): Promise<void> => {
            const client = getClient();
            return client.deleteTemplate(templateId);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: [TEMPLATES_QUERY_KEY] });
        },
    });
}

export function useDeployTemplate() {
    return useMutation({
        mutationFn: async (req: DeployTemplateRequest): Promise<{ plan_id: string; executions: any[] }> => {
            const client = getClient();
            return client.deployTemplate(req);
        },
    });
}

export function usePreviewTemplateDeployment() {
    return useMutation({
        mutationFn: async ({ templateId, siteId }: { templateId: string; siteId: string }): Promise<{ actions: any[] }> => {
            const client = getClient();
            return client.previewTemplateDeployment(templateId, siteId);
        },
    });
}

// Sites hooks

export function useSites() {
    return useQuery({
        queryKey: ['sites'],
        queryFn: async () => {
            const client = getClient();
            return client.listSites();
        },
    });
}
